package build

import (
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/runner"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
)

// Artifact represents a file that has been collected from a job.
type Artifact struct {
	ID        int64
	UserID    int64
	BuildID   int64
	JobID     int64
	Hash      string
	Source    string
	Name      string
	Size      sql.NullInt64
	MD5       []byte
	SHA256    []byte
	CreatedAt time.Time
	DeletedAt sql.NullTime

	Build *Build
	Job   *Job
	User  *user.User
}

var _ database.Model = (*Artifact)(nil)

func (a *Artifact) Dest() []interface{} {
	return []interface{}{
		&a.ID,
		&a.UserID,
		&a.BuildID,
		&a.JobID,
		&a.Hash,
		&a.Source,
		&a.Name,
		&a.Size,
		&a.MD5,
		&a.SHA256,
		&a.CreatedAt,
		&a.DeletedAt,
	}
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
	case *user.User:
		if a.UserID == v.ID {
			a.User = v
		}
	}
}

// JSON returns a map[string]interface{} representation of the current Artifact.
// This will include the User, Build, and Job models if they are non-nil.
func (a *Artifact) JSON(addr string) map[string]interface{} {
	if a == nil {
		return nil
	}

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
		"deleted_at": nil,
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

	if a.DeletedAt.Valid {
		json["deleted_at"] = a.DeletedAt.Time.Format(time.RFC3339)
	}

	if a.User != nil {
		json["user"] = a.User.JSON(addr)
	}

	if a.Build != nil {
		json["build"] = a.Build.JSON(addr)
	}

	if a.Job != nil {
		json["job"] = a.Job.JSON(addr)
	}
	return json
}

// Endpoint returns the endpoint for the current Artifact. this will only
// return an endpoint if the current Artifact has a non-nil build. The given
// uris are appended to the returned endpoint.
func (a *Artifact) Endpoint(uris ...string) string {
	if a.Build == nil {
		return ""
	}

	uris = append([]string{"artifacts", a.Name}, uris...)
	return a.Build.Endpoint(uris...)
}

// Values returns all of the values for the current Artifact.
func (a *Artifact) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":         a.ID,
		"user_id":    a.UserID,
		"build_id":   a.BuildID,
		"job_id":     a.JobID,
		"hash":       a.Hash,
		"source":     a.Source,
		"name":       a.Name,
		"size":       a.Size,
		"md5":        a.MD5,
		"sha256":     a.SHA256,
		"created_at": a.CreatedAt,
		"deleted_at": a.DeletedAt,
	}
}

// ArtifactStore allows for the retrieval of build Artifacts. This makes use of
// the fs.Store interface for collecting Artifacts from a build.
type ArtifactStore struct {
	database.Pool
	fs.Store
}

var (
	_ database.Loader  = (*ArtifactStore)(nil)
	_ runner.Collector = (*ArtifactStore)(nil)

	artifactTable = "build_artifacts"
)

// Get returns the singular build Artifact that can be found with the given
// query options applied, along with whether or not one could be found.
func (s *ArtifactStore) Get(opts ...query.Option) (*Artifact, bool, error) {
	var a Artifact

	ok, err := s.Pool.Get(artifactTable, &a, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &a, ok, nil
}

// All returns all of the build Artifacts that can be found with the given
// query options applied.
func (s *ArtifactStore) All(opts ...query.Option) ([]*Artifact, error) {
	aa := make([]*Artifact, 0)

	new := func() database.Model {
		a := &Artifact{}
		aa = append(aa, a)
		return a
	}

	if err := s.Pool.All(artifactTable, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return aa, nil
}

// Deleted marks all of the Artifacts in the given list of ids as deleted. This
// will not remove records from the table, but will simply zero-out the columns.
func (s *ArtifactStore) Deleted(ids ...int64) error {
	if len(ids) == 0 {
		return nil
	}

	vals := make([]interface{}, 0, len(ids))

	for _, id := range ids {
		vals = append(vals, id)
	}

	q := query.Update(
		artifactTable,
		query.Set("size", query.Arg(0)),
		query.Set("md5", query.Arg(nil)),
		query.Set("sha256", query.Arg(nil)),
		query.Set("deleted_at", query.Arg(time.Now())),
		query.Where("id", "IN", query.List(vals...)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

// collector is for collecting Artifacts for the specified build.
type collector struct {
	store   *ArtifactStore
	userId  int64
	buildId int64
}

func (c *collector) Collect(name string, r io.Reader) (int64, error) {
	part, err := c.store.Partition(c.userId)

	if err != nil {
		return 0, errors.Err(err)
	}

	a, ok, err := c.store.Get(
		query.Where("build_id", "=", query.Arg(c.buildId)),
		query.Where("name", "=", query.Arg(name)),
	)

	if err != nil {
		return 0, errors.Err(err)
	}

	if !ok {
		return 0, &fs.PathError{
			Op:   "collect",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	md5 := md5.New()
	sha256 := sha256.New()
	tee := io.TeeReader(r, io.MultiWriter(md5, sha256))

	n, err := part.Collect(a.Hash, tee)

	if err != nil {
		return 0, errors.Err(err)
	}

	q := query.Update(
		artifactTable,
		query.Set("size", query.Arg(n)),
		query.Set("md5", query.Arg(md5.Sum(nil))),
		query.Set("sha256", query.Arg(sha256.Sum(nil))),
		query.Where("id", "=", query.Arg(a.ID)),
	)

	if _, err := c.store.Exec(q.Build(), q.Args()...); err != nil {
		return 0, errors.Err(err)
	}
	return n, nil
}

// Collector will configure a collector for storing artifacts collected from
// the given build. The underlying store will be patitioned using the ID of the
// user who owns the given build.
func (s *ArtifactStore) Collector(b *Build) runner.Collector {
	return &collector{
		store:   s,
		userId:  b.UserID,
		buildId: b.ID,
	}
}

func (s *ArtifactStore) Load(fk, pk string, mm ...database.Model) error {
	vals := database.Values(fk, mm)

	aa, err := s.All(query.Where(pk, "IN", database.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	loaded := make([]database.Model, 0, len(aa))

	for _, a := range aa {
		loaded = append(loaded, a)
	}

	database.Bind(fk, pk, loaded, mm)
	return nil
}
