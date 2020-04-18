package build

import (
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"strings"
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

func NewArtifactStore(db *sqlx.DB, mm ...model.Model) ArtifactStore {
	s := ArtifactStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func NewArtifactStoreWithCollector(db *sqlx.DB, c runner.Collector, mm ...model.Model) ArtifactStore {
	s := ArtifactStore{
		Store:     model.Store{DB: db},
		collector: c,
	}
	s.Bind(mm...)
	return s
}

func ArtifactModel(aa []*Artifact) func(int) model.Model {
	return func(i int) model.Model {
		return aa[i]
	}
}

func (a *Artifact) Kind() string { return "build_artifact" }

func (a *Artifact) Bind(mm ...model.Model) {
	if a == nil {
		return
	}

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
	if a == nil {
		return
	}
	a.ID = id
}

func (a *Artifact) Primary() (string, int64) {
	if a == nil {
		return "id", 0
	}
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

func (a *Artifact) Endpoint(uri ...string) string {
	if a == nil {
		return ""
	}
	if a.Build == nil || a.Build.IsZero() {
		return ""
	}

	uri = append([]string{"artifacts", fmt.Sprintf("%v", a.ID)}, uri...)
	return a.Build.Endpoint(uri...)
}

func (a *Artifact) Values() map[string]interface{} {
	if a == nil {
		return map[string]interface{}{}
	}

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

func (s ArtifactStore) Load(key string, vals []interface{}, fn model.LoaderFunc) error {
	return nil
}

func (s ArtifactStore) Create(aa ...*Artifact) error {
	models := model.Slice(len(aa), ArtifactModel(aa))
	return errors.Err(s.Store.Create(artifactTable, models...))
}

func (s ArtifactStore) Update(aa ...*Artifact) error {
	models := model.Slice(len(aa), ArtifactModel(aa))
	return errors.Err(s.Store.Update(artifactTable, models...))
}

func (s ArtifactStore) New() *Artifact {
	a := &Artifact{
		Build: s.Build,
		Job:   s.Job,
	}

	if s.Build != nil {
		_, id := s.Build.Primary()
		a.BuildID = id
	}

	if s.Job != nil {
		_, id := s.Job.Primary()
		a.JobID = id
	}
	return a
}

func (s ArtifactStore) All(opts ...query.Option) ([]*Artifact, error) {
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

func (s ArtifactStore) Get(opts ...query.Option) (*Artifact, error) {
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

func (s ArtifactStore) Collect(name string, r io.Reader) (int64, error) {
	if s.collector == nil {
		return 0, errors.New("cannot collect artifact: nil collector")
	}

	md5 := md5.New()
	sha256 := sha256.New()
	tee := io.TeeReader(r, io.MultiWriter(md5, sha256))

	n, err := s.collector.Collect(name, tee)

	if err != nil {
		return n, errors.Err(err)
	}

	a, err := s.Get(query.Where("hash", "=", strings.TrimSuffix(name, ".tar")))

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
