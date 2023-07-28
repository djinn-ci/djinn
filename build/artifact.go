package build

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"

	"github.com/andrewpillar/fs"
	"github.com/andrewpillar/query"
)

// Artifact represents a file that has been collected from a job.
type Artifact struct {
	loaded    []string
	ID        int64
	UserID    int64
	BuildID   int64
	JobID     int64
	Hash      string
	Source    string
	Name      string
	Size      database.Null[int64]
	MD5       database.Bytea
	SHA256    database.Bytea
	CreatedAt time.Time
	DeletedAt database.Null[time.Time]

	Build *Build
	Job   *Job
	User  *auth.User
}

var _ database.Model = (*Artifact)(nil)

func (a *Artifact) Primary() (string, any) { return "id", a.ID }

func (a *Artifact) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":         &a.ID,
		"user_id":    &a.UserID,
		"build_id":   &a.BuildID,
		"job_id":     &a.JobID,
		"hash":       &a.Hash,
		"source":     &a.Source,
		"name":       &a.Name,
		"size":       &a.Size,
		"md5":        &a.MD5,
		"sha256":     &a.SHA256,
		"created_at": &a.CreatedAt,
		"deleted_at": &a.DeletedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (a *Artifact) Params() database.Params {
	params := database.Params{
		"id":         database.ImmutableParam(a.ID),
		"user_id":    database.CreateOnlyParam(a.UserID),
		"build_id":   database.CreateOnlyParam(a.BuildID),
		"job_id":     database.CreateOnlyParam(a.JobID),
		"hash":       database.CreateOnlyParam(a.Hash),
		"source":     database.CreateOnlyParam(a.Source),
		"name":       database.CreateOnlyParam(a.Name),
		"size":       database.UpdateOnlyParam(a.Size),
		"md5":        database.UpdateOnlyParam(a.MD5),
		"sha256":     database.UpdateOnlyParam(a.SHA256),
		"created_at": database.CreateOnlyParam(a.CreatedAt),
		"deleted_at": database.UpdateOnlyParam(a.DeletedAt),
	}

	if len(a.loaded) > 0 {
		params.Only(a.loaded...)
	}
	return params
}

// Endpoint returns the endpoint for the current Artifact. this will only
// return an endpoint if the current Artifact has a non-nil build. The given
// elems are appended to the returned endpoint.
func (a *Artifact) Endpoint(elems ...string) string {
	if a.Build == nil {
		return ""
	}

	elems = append([]string{"artifacts", a.Name}, elems...)
	return a.Build.Endpoint(elems...)
}

// Bind the given Model to the current Artifact if it is one of Build, Job, or
// User, and if there is a direct relation between the two.
func (a *Artifact) Bind(m database.Model) {
	switch v := m.(type) {
	case *Build:
		if a.BuildID == v.ID {
			a.Build = v
		}
	case *Job:
		if a.JobID == v.ID {
			a.Job = v
		}
	case *auth.User:
		if a.UserID == v.ID {
			a.User = v
		}
	}
}

func (a *Artifact) MarshalJSON() ([]byte, error) {
	if a == nil {
		return []byte("null"), nil
	}

	b, err := json.Marshal(map[string]any{
		"user_id":    a.UserID,
		"build_id":   a.BuildID,
		"job_id":     a.JobID,
		"source":     a.Source,
		"name":       a.Name,
		"size":       a.Size,
		"md5":        a.MD5,
		"sha256":     a.SHA256,
		"created_at": a.CreatedAt,
		"deleted_at": a.DeletedAt,
		"url":        env.DJINN_API_SERVER + a.Endpoint(),
		"user":       a.User,
		"build":      a.Build,
		"job":        a.Job,
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (a *Artifact) filestore(store fs.FS) (fs.FS, error) {
	store, err := store.Sub(strconv.FormatInt(a.UserID, 10))

	if err != nil {
		return nil, errors.Err(err)
	}
	return store, nil
}

func (a *Artifact) Open(store fs.FS) (fs.File, error) {
	store, err := a.filestore(store)

	if err != nil {
		return nil, errors.Err(err)
	}

	f, err := store.Open(a.Hash)

	if err != nil {
		return nil, errors.Err(err)
	}
	return f, nil
}

const artifactTable = "build_artifacts"

type ArtifactStore struct {
	*database.Store[*Artifact]

	FS     fs.FS
	Hasher *crypto.Hasher
}

func NewArtifactStore(pool *database.Pool) *database.Store[*Artifact] {
	return database.NewStore[*Artifact](pool, artifactTable, func() *Artifact {
		return &Artifact{}
	})
}

func (s *ArtifactStore) CreateTx(ctx context.Context, tx database.Tx, a *Artifact) error {
	hash, err := s.Hasher.HashNow()

	if err != nil {
		return errors.Err(err)
	}

	a.Hash = hash

	if err := s.Store.CreateTx(ctx, tx, a); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *ArtifactStore) Index(ctx context.Context, vals url.Values, opts ...query.Option) (*database.Paginator[*Artifact], error) {
	page, _ := strconv.Atoi(vals.Get("page"))

	opts = append(opts,
		database.Search("name", vals.Get("search")),
	)

	paginator, err := s.Paginate(ctx, page, database.PageLimit, opts...)

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := paginator.Load(ctx, s.Store, append(opts, query.OrderAsc("name"))...); err != nil {
		return nil, errors.Err(err)
	}
	return paginator, nil
}

func (s *ArtifactStore) Delete(ctx context.Context, aa ...*Artifact) error {
	if len(aa) == 0 {
		return nil
	}

	ids := database.Map[*Artifact, any](aa, func(a *Artifact) any {
		return a.ID
	})

	a := Artifact{
		loaded: []string{"size", "md5", "sha256", "deleted_at"},
		Size:   database.Null[int64]{},
		MD5:    nil,
		SHA256: nil,
		DeletedAt: database.Null[time.Time]{
			Elem:  time.Now(),
			Valid: true,
		},
	}

	if err := s.UpdateMany(ctx, &a, query.Where("id", "IN", query.List(ids...))); err != nil {
		return errors.Err(err)
	}

	hashes := database.Map[*Artifact, string](aa, func(a *Artifact) string {
		return a.Hash
	})

	for _, a := range aa {
		store, err := a.filestore(s.FS)

		if err != nil {
			return errors.Err(err)
		}

		for _, hash := range hashes {
			if err := store.Remove(hash); err != nil {
				if !errors.Is(err, fs.ErrNotExist) {
					return errors.Err(err)
				}
			}
		}
	}
	return nil
}

type artifactFilestore struct {
	fs.FS

	limit int64
	store *ArtifactStore
	build *Build
}

func (s *artifactFilestore) Put(f fs.File) (fs.File, error) {
	info, err := f.Stat()

	if err != nil {
		return nil, errors.Err(err)
	}

	name := info.Name()

	ctx := context.Background()

	a, ok, err := s.store.SelectOne(
		ctx,
		[]string{"id", "user_id", "hash", "size", "md5", "sha256"},
		query.Where("build_id", "=", query.Arg(s.build.ID)),
		query.Where("name", "=", query.Arg(name)),
	)

	if err != nil {
		return nil, &fs.PathError{Op: "put", Path: name, Err: err}
	}

	if !ok {
		return nil, &fs.PathError{Op: "put", Path: name, Err: fs.ErrNotExist}
	}

	md5 := md5.New()
	sha256 := sha256.New()

	var r io.Reader = f

	if s.limit > 0 {
		r = io.LimitReader(f, s.limit)
	}

	f, err = fs.ReadFile(a.Hash, io.TeeReader(r, io.MultiWriter(md5, sha256)))

	if err != nil {
		return nil, &fs.PathError{Op: "put", Path: name, Err: err}
	}

	defer fs.Cleanup(f)

	store, err := a.filestore(s.store.FS)

	if err != nil {
		return nil, &fs.PathError{Op: "put", Path: name, Err: err}
	}

	f, err = store.Put(f)

	if err != nil {
		return nil, &fs.PathError{Op: "put", Path: name, Err: err}
	}

	a.Size.Elem = info.Size()
	a.Size.Valid = true
	a.MD5 = database.Bytea(md5.Sum(nil))
	a.SHA256 = database.Bytea(sha256.Sum(nil))

	if err := s.store.Update(ctx, a); err != nil {
		return nil, &fs.PathError{Op: "put", Path: name, Err: err}
	}
	return f, nil
}

func (s *ArtifactStore) Filestore(b *Build, limit int64) fs.FS {
	return fs.WriteOnly(&artifactFilestore{
		FS:    s.FS,
		limit: limit,
		store: s,
		build: b,
	})
}
