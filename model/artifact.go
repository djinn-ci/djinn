package model

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"
)

type Artifact struct {
	model

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
	stmt, err := a.Prepare(`
		INSERT INTO artifacts (build_id, job_id, hash, source, name)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(a.BuildID, a.JobID, a.Hash, a.Source, a.Name)

	return errors.Err(row.Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt))
}

func (a Artifact) IsZero() bool {
	return a.model.IsZero() &&
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
	stmt, err := a.Prepare(`
		UPDATE artifacts
		SET size = $1, md5 = $2, sha256 = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(a.Size, a.MD5, a.SHA256, a.ID)

	return errors.Err(row.Scan(&a.UpdatedAt))
}

func (as ArtifactStore) All() ([]*Artifact, error) {
	aa := make([]*Artifact, 0)

	query := "SELECT * FROM artifacts"
	args := []interface{}{}

	if as.Build != nil {
		query += " WHERE build_id = $1"
		args = append(args, as.Build.ID)
	}

	if as.Job != nil {
		if as.Build != nil {
			query += " AND job_id = $2"
		} else {
			query += " WHERE job_id = $1"
		}

		args = append(args, as.Job.ID)
	}

	err := as.Select(&aa, query, args...)

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

func (as ArtifactStore) Find(id int64) (*Artifact, error) {
	a := &Artifact{
		model: model{
			DB: as.DB,
		},
		Build: as.Build,
		Job:   as.Job,
	}

	query := "SELECT * FROM artifacts WHERE id = $1"
	args := []interface{}{id}

	if as.Build != nil {
		query += " AND build_id = $2"
		args = append(args, as.Build.ID)
	}

	if as.Job != nil {
		if as.Build != nil {
			query += " AND job_id = $3"
		} else {
			query += " AND job_id = $2"
		}

		args = append(args, as.Job.ID)
	}

	err := as.Get(a, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return a, errors.Err(err)
}

func (as ArtifactStore) FindByHash(hash string) (*Artifact, error) {
	a := &Artifact{
		model: model{
			DB: as.DB,
		},
		Build: as.Build,
		Job:   as.Job,
	}

	query := "SELECT * FROM artifacts WHERE hash = $1"
	args := []interface{}{hash}

	if as.Build != nil {
		query += " AND build_id = $2"
		args = append(args, as.Build.ID)
	}

	if as.Job != nil {
		if as.Build != nil {
			query += " AND job_id = $3"
		} else {
			query += " AND job_id = $2"
		}

		args = append(args, as.Job.ID)
	}

	err := as.Get(a, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return a, errors.Err(err)
}

func (as ArtifactStore) InJobID(ids ...int64) ([]*Artifact, error) {
	aa := make([]*Artifact, 0)

	query, args, err := sqlx.In("SELECT * FROM artifacts WHERE job_id IN (?)", ids)

	if err != nil {
		return aa, errors.Err(err)
	}

	err = as.Select(&aa, as.Rebind(query), args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, a := range aa {
		a.DB = as.DB
	}

	return aa, errors.Err(err)
}

func (as ArtifactStore) New() *Artifact {
	a := &Artifact{
		model: model{
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
