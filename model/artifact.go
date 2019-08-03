package model

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"

	"github.com/jmoiron/sqlx"
)

type Artifact struct {
	Model

	BuildID int64          `db:"build_id"`
	JobID   int64          `db:"job_id"`
	Hash    string         `db:"hash"`
	Source  string         `db:"source"`
	Name    string         `db:"name"`
	Size    sql.NullInt64  `db:"size"`
	MD5     []byte         `db:"md5"`
	SHA256  []byte         `db:"sha256"`

	Build *Build
	Job   *Job
}

type ArtifactStore struct {
	*sqlx.DB

	Build *Build
	Job   *Job
}

func (a *Artifact) Create() error {
	q := query.Insert(
		query.Table("artifacts"),
		query.Columns("build_id", "job_id", "hash", "source", "name"),
		query.Values(a.BuildID, a.JobID, a.Hash, a.Source, a.Name),
		query.Returning("id", "created_at", "updated_at"),
	)

	stmt, err := a.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt))
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

func (a Artifact) UIEndpoint(uri ...string) string {
	endpoint := fmt.Sprintf("/builds/%v/artifacts/%v", a.BuildID, a.ID)

	if len(uri) > 0 {
		endpoint = fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}

	return endpoint
}

func (a *Artifact) Update() error {
	q := query.Update(
		query.Table("artifacts"),
		query.Set("size", a.Size),
		query.Set("md5", a.MD5),
		query.Set("sha256", a.SHA256),
		query.SetRaw("updated_at", "NOW()"),
		query.WhereEq("id", a.ID),
		query.Returning("updated_at"),
	)

	stmt, err := a.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&a.UpdatedAt))
}

func (as ArtifactStore) All(opts ...query.Option) ([]*Artifact, error) {
	aa := make([]*Artifact, 0)

	opts = append([]query.Option{query.Columns("*")}, opts...)
	opts = append(opts, ForBuild(as.Build), ForJob(as.Job), query.Table("artifacts"))

	q := query.Select(opts...)

	err := as.Select(&aa, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, a := range aa {
		a.DB = as.DB
		a.Build = as.Build
		a.Job = as.Job
	}

	return aa, errors.Err(err)
}

func (as ArtifactStore) findBy(col string, val interface{}) (*Artifact, error) {
	a := &Artifact{
		Model: Model{
			DB: as.DB,
		},
		Build: as.Build,
		Job:   as.Job,
	}

	q := query.Select(
		query.Columns("*"),
		query.Table("artifacts"),
		query.WhereEq(col, val),
		ForBuild(as.Build),
		ForJob(as.Job),
	)

	err := as.Get(a, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return a, errors.Err(err)
}

func (as ArtifactStore) Find(id int64) (*Artifact, error) {
	a, err := as.findBy("id", id)

	return a, errors.Err(err)
}

func (as ArtifactStore) FindByHash(hash string) (*Artifact, error) {
	a, err := as.findBy("hash", hash)

	return a, errors.Err(err)
}

func (as ArtifactStore) Index(opts ...query.Option) ([]*Artifact, error) {
	aa, err := as.All(opts...)

	return aa, errors.Err(err)
}

func (as ArtifactStore) New() *Artifact {
	a := &Artifact{
		Model: Model{
			DB: as.DB,
		},
		Build: as.Build,
		Job:   as.Job,
	}

	if as.Build != nil {
		a.BuildID = as.Build.ID
	}

	if as.Job != nil {
		a.JobID = as.Job.ID
	}

	return a
}
