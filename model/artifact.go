package model

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"
)

type Artifact struct {
	Model

	BuildID int64         `db:"build_id"`
	JobID   int64         `db:"job_id"`
	Hash    string        `db:"hash"`
	Source  string        `db:"source"`
	Name    string        `db:"name"`
	Size    sql.NullInt64 `db:"size"`
	MD5     []byte        `db:"md5"`
	SHA256  []byte        `db:"sha256"`

	Build *Build
	Job   *Job
}

type ArtifactStore struct {
	Store

	Build *Build
	Job   *Job
}

func (a Artifact) IsZero() bool {
	return a.Model.IsZero() &&
		a.BuildID == 0 &&
		a.JobID == 0 &&
		a.Hash == "" &&
		a.Source == "" &&
		a.Name == "" &&
		!a.Size.Valid &&
		len(a.MD5) == 0 &&
		len(a.SHA256) == 0
}

func (a Artifact) Values() map[string]interface{} {
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

func (a Artifact) UIEndpoint(uri ...string) string {
	endpoint := fmt.Sprintf("/builds/%v/artifacts/%v", a.BuildID, a.ID)

	if len(uri) > 0 {
		endpoint = fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}

	return endpoint
}

func (s ArtifactStore) All(opts ...query.Option) ([]*Artifact, error) {
	aa := make([]*Artifact, 0)

	opts = append(opts, ForBuild(s.Build), ForJob(s.Job))

	err := s.Store.All(&aa, ArtifactTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, a := range aa {
		a.DB = s.DB
		a.Build = s.Build
		a.Job = s.Job
	}

	return aa, errors.Err(err)
}

func (s ArtifactStore) Create(aa ...*Artifact) error {
	return errors.Err(s.Store.Create(ArtifactTable, s.interfaceSlice(aa...)...))
}

func (s ArtifactStore) findBy(col string, val interface{}) (*Artifact, error) {
	a := &Artifact{
		Model: Model{
			DB: s.DB,
		},
	}

	err := s.FindBy(a, ArtifactTable, col, val)

	return a, errors.Err(err)
}

func (s ArtifactStore) Find(id int64) (*Artifact, error) {
	a, err := s.findBy("id", id)

	return a, errors.Err(err)
}

func (s ArtifactStore) FindByHash(hash string) (*Artifact, error) {
	a, err := s.findBy("hash", hash)

	return a, errors.Err(err)
}

func (s ArtifactStore) Index(opts ...query.Option) ([]*Artifact, error) {
	aa, err := s.All(opts...)

	return aa, errors.Err(err)
}

func (s ArtifactStore) interfaceSlice(aa ...*Artifact) []Interface {
	ii := make([]Interface, len(aa), len(aa))

	for i, a := range aa {
		ii[i] = a
	}

	return ii
}

func (s ArtifactStore) New() *Artifact {
	a := &Artifact{
		Model: Model{
			DB: s.DB,
		},
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

func (s ArtifactStore) Update(aa ...*Artifact) error {
	return errors.Err(s.Store.Update(ArtifactTable, s.interfaceSlice(aa...)...))
}
