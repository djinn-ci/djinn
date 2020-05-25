package build

import (
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/runner"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type Artifact struct {
	ID        int64         `db:"id"`
	BuildID   int64         `db:"build_id"`
	JobID     int64         `db:"job_id"`
	Hash      string        `db:"hash"`
	Source    string        `db:"source"`
	Name      string        `db:"name"`
	Size      sql.NullInt64 `db:"size"`
	MD5       []byte        `db:"md5"`
	SHA256    []byte        `db:"sha256"`
	CreatedAt time.Time     `db:"created_at"`

	Build *Build `db:"-"`
	Job   *Job   `db:"-"`
}

type ArtifactStore struct {
	model.Store

	collector runner.Collector
	Build     *Build
	Job       *Job
}

var (
	_ model.Model      = (*Artifact)(nil)
	_ model.Binder     = (*ArtifactStore)(nil)
	_ model.Loader     = (*ArtifactStore)(nil)
	_ runner.Collector = (*ArtifactStore)(nil)

	artifactTable = "build_artifacts"
)

// NewArtifactStore returns a new ArtifactStore for querying the build_artifacts
// table. Each model passed to this function will be bound to the returned
// ArtifactStore.
func NewArtifactStore(db *sqlx.DB, mm ...model.Model) *ArtifactStore {
	s := &ArtifactStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// NewArtifactStoreWithCollector returns a new ArtifactStore with the given
// runner.Collector. This allows for the ArtifactStore to be used as a
// runner.Collector during job execution. Each collected artifact will be
// updated in the database, with the actual collection being deferred to the
// given runner.Collector.
func NewArtifactStoreWithCollector(db *sqlx.DB, c runner.Collector, mm ...model.Model) *ArtifactStore {
	s := NewArtifactStore(db, mm...)
	s.collector = c
	return s
}

// ArtifactModel is called along with model.Slice to convert the given slice of
// Artifact models to a slice of model.Model interfaces.
func ArtifactModel(aa []*Artifact) func(int) model.Model {
	return func(i int) model.Model {
		return aa[i]
	}
}

// Bind the given models to the current Artifact. This will only bind the model if
// they are one of the following,
//
// - *Build
// - *Job
func (a *Artifact) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			a.Build = m.(*Build)
		case *Job:
			a.Job = m.(*Job)
		}
	}
}

func (a *Artifact) SetPrimary(id int64) {
	a.ID = id
}

func (a *Artifact) Primary() (string, int64) {
	return "id", a.ID
}

func (a *Artifact) IsZero() bool {
	return a == nil || a.ID == 0 &&
		a.BuildID == 0 &&
		a.JobID == 0 &&
		a.Hash == "" &&
		a.Source == "" &&
		a.Name == "" &&
		!a.Size.Valid &&
		len(a.MD5) == 0 &&
		len(a.SHA256) == 0
}

func (a *Artifact) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":         a.ID,
		"build_id":   a.BuildID,
		"job_id":     a.JobID,
		"source":     a.Source,
		"name":       a.Name,
		"size":       nil,
		"md5":        fmt.Sprintf("%x", a.MD5),
		"sha256":     fmt.Sprintf("%x", a.SHA256),
		"created_at": a.CreatedAt.Format(time.RFC3339),
		"url":        addr + a.Endpoint(),
	}

	if a.Size.Valid {
		json["size"] = a.Size.Int64
	}

	for name, m := range map[string]model.Model{
		"build": a.Build,
		"job":   a.Job,
	}{
		if !m.IsZero() {
			json[name] = m.JSON(addr)
		}
	}
	return json
}

// Endpoint returns the endpoint for the current Artifact. If the bound Build
// model is nil, then an empty string is returned, otherwise the endpoint is
// prefixed with the Build's endpoint.
func (a *Artifact) Endpoint(_ ...string) string {
	if a.Build == nil || a.Build.IsZero() {
		return ""
	}
	return a.Build.Endpoint("artifacts", strconv.FormatInt(a.ID, 10))
}

func (a *Artifact) Values() map[string]interface{} {
	return map[string]interface{}{
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

// Bind the given models to the current ArtifactStore. This will only bind the
// model if they are one of the following,
//
// - *Build
// - *Job
func (s *ArtifactStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			s.Build = m.(*Build)
		case *Job:
			s.Job = m.(*Job)
		}
	}
}

// Load gets a slice of Artifact models. The given key, and vals are applied to
// the underlying query as a WHERE IN clause, like so,
//
//   WHERE key IN (vals,...)
//
// each model in the slice is then loaded via the given callback. Any models
// that are bound to the ArtifactStore will be applied via model.Where during
// querying.
func (s *ArtifactStore) Load(key string, vals []interface{}, load model.LoaderFunc) error {
	aa, err := s.All(query.Where(key, "IN", vals...))

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

// Create inserts the given Artifact models into the build_artifacts table.
func (s *ArtifactStore) Create(aa ...*Artifact) error {
	models := model.Slice(len(aa), ArtifactModel(aa))
	return errors.Err(s.Store.Create(artifactTable, models...))
}

// Update updates the given Artifact models in the build_artifacts table.
func (s *ArtifactStore) Update(aa ...*Artifact) error {
	models := model.Slice(len(aa), ArtifactModel(aa))
	return errors.Err(s.Store.Update(artifactTable, models...))
}

// New returns a new Artifact binding any non-nil models to it from the current
// ArtifactStore.
func (s *ArtifactStore) New() *Artifact {
	a := &Artifact{
		Build: s.Build,
		Job:   s.Job,
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
// given. Each model that is bound to the store will be applied to the list of
// query options via model.Where. For example, if a Build model is bound to a
// store then the following WHERE clause would be applied to the query,
//
//   WHERE build_id = s.Build.ID
func (s *ArtifactStore) All(opts ...query.Option) ([]*Artifact, error) {
	aa := make([]*Artifact, 0)

	opts = append([]query.Option{
		model.Where(s.Build, "build_id"),
		model.Where(s.Job, "job_id"),
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
// given. The model.Where option is used on the Build and Job bound models to
// limit the query to those relations. For example, if a Build model is bound
// to a store then the following WHERE clause would be applied to the query,
//
//   WHERE build_id = s.Build.ID
func (s *ArtifactStore) Get(opts ...query.Option) (*Artifact, error) {
	a := &Artifact{
		Build: s.Build,
		Job:   s.Job,
	}

	opts = append([]query.Option{
		model.Where(s.Build, "build_id"),
		model.Where(s.Job, "job_id"),
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

	a, err := s.Get(query.Where("name", "=", name))

	if err != nil {
		return 0, errors.Err(err)
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

	a.Size = sql.NullInt64{
		Int64: n,
		Valid: true,
	}
	a.MD5 = md5.Sum(nil)
	a.SHA256 = sha256.Sum(nil)

	err = s.Update(a)
	return n, errors.Err(err)
}
