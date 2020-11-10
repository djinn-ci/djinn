package build

import (
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/runner"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Artifact is the type that represents a file that has been collected from a
// build environment for a given build. This contains metadata about the file
// that was collected.
type Artifact struct {
	ID        int64         `db:"id"`
	UserID    int64         `db:"user_id"`
	BuildID   int64         `db:"build_id"`
	JobID     int64         `db:"job_id"`
	Hash      string        `db:"hash"`
	Source    string        `db:"source"`
	Name      string        `db:"name"`
	Size      sql.NullInt64 `db:"size"`
	MD5       []byte        `db:"md5"`
	SHA256    []byte        `db:"sha256"`
	CreatedAt time.Time     `db:"created_at"`

	Build *Build     `db:"-"`
	Job   *Job       `db:"-"`
	User  *user.User `db:"-"`
}

// ArtifactStore is the type for creating and modifying Artifact models in the
// database. The ArtifactStore type can have an underlying runner.Collector
// implementation that can allow for it to be used for collecting artifacts
// from a build environment.
type ArtifactStore struct {
	database.Store

	collector runner.Collector

	// Build is the bound Build model. If not nil this will bind the Build
	// model to any Artifact models that are created. If not nil this will
	// append a WHERE clause on the build_id column for all SELECT queries
	// performed.
	Build *Build

	// Job is the bound Job model. If not nil this will bind the Job model to
	// any Artifact models that are created. If not nil this will append a
	// WHERE clause on the job_id column for all SELECT queries performed.
	Job *Job

	User *user.User
}

var (
	_ database.Model   = (*Artifact)(nil)
	_ database.Binder  = (*ArtifactStore)(nil)
	_ database.Loader  = (*ArtifactStore)(nil)
	_ runner.Collector = (*ArtifactStore)(nil)

	artifactTable = "build_artifacts"
)

// NewArtifactStore returns a new ArtifactStore for querying the build_artifacts
// table. Each model passed to this function will be bound to the returned
// ArtifactStore.
func NewArtifactStore(db *sqlx.DB, mm ...database.Model) *ArtifactStore {
	s := &ArtifactStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// NewArtifactStoreWithCollector returns a new ArtifactStore with the given
// runner.Collector. This allows for the ArtifactStore to be used as a
// runner.Collector during a build run. Each collected artifact will be
// updated in the database, with the actual collection being deferred to the
// given runner.Collector.
func NewArtifactStoreWithCollector(db *sqlx.DB, c runner.Collector, mm ...database.Model) *ArtifactStore {
	s := NewArtifactStore(db, mm...)
	s.collector = c
	return s
}

// ArtifactModel is called along with database.ModelSlice to convert the given slice of
// Artifact models to a slice of database.Model interfaces.
func ArtifactModel(aa []*Artifact) func(int) database.Model {
	return func(i int) database.Model {
		return aa[i]
	}
}

// Bind implements the database.Binder interface. This will only bind the models
// if they are pointers to either Build or Job models.
func (a *Artifact) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			a.Build = m.(*Build)
		case *Job:
			a.Job = m.(*Job)
		case *user.User:
			a.User = m.(*user.User)
		}
	}
}

// SetPrimary implements the database.Model interface.
func (a *Artifact) SetPrimary(id int64) { a.ID = id }

// Primary implements the database.Model interface.
func (a *Artifact) Primary() (string, int64) { return "id", a.ID }

// IsZero implements the database.Model interface.
func (a *Artifact) IsZero() bool {
	return a == nil || a.ID == 0 &&
		a.UserID == 0 &&
		a.BuildID == 0 &&
		a.JobID == 0 &&
		a.Hash == "" &&
		a.Source == "" &&
		a.Name == "" &&
		!a.Size.Valid &&
		len(a.MD5) == 0 &&
		len(a.SHA256) == 0
}

// JSON implements the database.Model interface. This will return a map with the
// current Artifacts values under each key. If any of the Build, or Job bound
// models exist on the Artifact, then the JSON representation of these models
// will be in the returned map, under the build, and job keys respectively.
func (a *Artifact) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":         a.ID,
		"user_id":    a.UserID,
		"build_id":   a.BuildID,
		"job_id":     a.JobID,
		"source":     a.Source,
		"name":       a.Name,
		"size":       nil,
		"md5":        nil,
		"sha256":     nil,
		"created_at": a.CreatedAt.Format(time.RFC3339),
		"url":        addr + a.Endpoint(),
	}

	if a.Size.Valid {
		json["size"] = a.Size.Int64
	}

	if len(a.MD5) > 0 {
		json["md5"] = fmt.Sprintf("%x", a.MD5)
	}
	if len(a.SHA256) > 0 {
		json["sha256"] = fmt.Sprintf("%x", a.SHA256)
	}

	for name, m := range map[string]database.Model{
		"user":  a.User,
		"build": a.Build,
		"job":   a.Job,
	} {
		if !m.IsZero() {
			json[name] = m.JSON(addr)
		}
	}
	return json
}

// Endpoint implements the database.Model interface. If the current Artifact
// has a nil or zero value Build bound model then an empty string is returned,
// otherwise the fulld Build endpoint is returned, suffixed with the Artifact
// endpoint, for example,
//
//   /b/l.belardo/10/artifacts/3
func (a *Artifact) Endpoint(uris ...string) string {
	if a.Build == nil || a.Build.IsZero() {
		return ""
	}
	uris = append([]string{"artifacts", strconv.FormatInt(a.ID, 10)}, uris...)
	return a.Build.Endpoint(uris...)
}

// Values implements the database.Model interface. This will return a map with
// the following values, user_id, build_id, job_id, hash, source, name, size,
// md5, and sha256.
func (a *Artifact) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":  a.UserID,
		"build_id": a.BuildID,
		"job_id":   a.JobID,
		"hash":     a.Hash,
		"source":   a.Source,
		"name":     a.Name,
		"size":     a.Size,
		"md5":      a.MD5,
		"sha256":   a.SHA256,
	}
}

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either Build or Job models.
func (s *ArtifactStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			s.Build = m.(*Build)
		case *Job:
			s.Job = m.(*Job)
		case *user.User:
			s.User = m.(*user.User)
		}
	}
}

// Load implements the database.Loader interface. Any models that are bound to
// the ArtifactStore will be applied during querying.
func (s *ArtifactStore) Load(key string, vals []interface{}, load database.LoaderFunc) error {
	aa, err := s.All(query.Where(key, "IN", query.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, a := range aa {
			load(i, a)
		}
	}
	return nil
}

// Create creates a new Artifact model in the database. The given hash should
// be unique across all Artifact models created. The src should be the verbatim
// name of the Artifact from the build environment. The dst should be the name
// that is used for collecting the Artifact.
func (s *ArtifactStore) Create(hash, src, dst string) (*Artifact, error) {
	a := s.New()
	a.Hash = hash
	a.Source = src
	a.Name = dst

	err := s.Store.Create(artifactTable, a)
	return a, errors.Err(err)
}

// New returns a new Artifact binding any non-nil models to it from the current
// ArtifactStore.
func (s *ArtifactStore) New() *Artifact {
	a := &Artifact{
		User:  s.User,
		Build: s.Build,
		Job:   s.Job,
	}

	if s.User != nil {
		a.UserID = s.User.ID
	}

	if s.Build != nil {
		a.BuildID = s.Build.ID
	}

	if s.Job != nil {
		a.JobID = s.Job.ID
	}
	return a
}

// All returns a slice of Artifact models, applying each query.Option that is
// given.
func (s *ArtifactStore) All(opts ...query.Option) ([]*Artifact, error) {
	aa := make([]*Artifact, 0)

	opts = append([]query.Option{
		database.Where(s.User, "user_id"),
		database.Where(s.Build, "build_id"),
		database.Where(s.Job, "job_id"),
	}, opts...)

	err := s.Store.All(&aa, artifactTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, a := range aa {
		a.Build = s.Build
		a.Job = s.Job
	}
	return aa, errors.Err(err)
}

// Get returns a single Artifact model, applying each query.Option that is
// given.
func (s *ArtifactStore) Get(opts ...query.Option) (*Artifact, error) {
	a := &Artifact{
		Build: s.Build,
		Job:   s.Job,
	}

	opts = append([]query.Option{
		database.Where(s.User, "user_id"),
		database.Where(s.Build, "build_id"),
		database.Where(s.Job, "job_id"),
	}, opts...)

	err := s.Store.Get(a, artifactTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return a, errors.Err(err)
}

// Collect looks up the Artifact by the given name, and updates it with the
// size, md5, and sha256 once the underlying runner.Collector has been
// successfully invoked. If no underlying collector has been set for the
// ArtifactStore then it immediately errors.
func (s *ArtifactStore) Collect(name string, r io.Reader) (int64, error) {
	if s.collector == nil {
		return 0, errors.New("cannot collect artifact: nil collector")
	}

	a, err := s.Get(query.Where("name", "=", query.Arg(name)))

	if err != nil {
		return 0, errors.Err(err)
	}

	if a.IsZero() {
		return 0, nil
	}

	md5 := md5.New()
	sha256 := sha256.New()
	tee := io.TeeReader(r, io.MultiWriter(md5, sha256))

	n, err := s.collector.Collect(a.Hash, tee)

	if errors.Cause(err) == io.EOF {
		err = nil
	}

	if err != nil {
		return n, errors.Err(err)
	}

	q := query.Update(
		artifactTable,
		query.Set("size", query.Arg(n)),
		query.Set("md5", query.Arg(md5.Sum(nil))),
		query.Set("sha256", query.Arg(sha256.Sum(nil))),
		query.Where("id", "=", query.Arg(a.ID)),
	)

	_, err = s.DB.Exec(q.Build(), q.Args()...)
	return n, errors.Err(err)
}
