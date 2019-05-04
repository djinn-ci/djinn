package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"
)

type Artifact struct {
	model

	BuildID int64          `db:"build_id"`
	JobID   int64          `db:"job_id"`
	Source  string         `db:"source"`
	Name    string         `db:"name"`
	Size    sql.NullInt64  `db:"size"`
	Type    sql.NullString `db:"type"`
	MD5     []byte         `db:"md5"`
	SHA256  []byte         `db:"sha256"`

	Build *Build
	Job   *Job
}

type ArtifactStore struct {
	*Store

	build *Build
	job   *Job
}

func (as ArtifactStore) New() *Artifact {
	a := &Artifact{
		model: model{
			DB: as.DB,
		},
		Build: as.build,
		Job:   as.job,
	}

	if as.build != nil {
		a.BuildID = as.build.ID
	}

	if as.job != nil {
		a.JobID = as.job.ID
	}

	return a
}

func (as ArtifactStore) Find(id int64) (*Artifact, error) {
	a := &Artifact{
		model: model{
			DB: as.DB,
		},
	}

	query := "SELECT * FROM artifacts WHERE id = $1"
	args := []interface{}{id}

	if as.build != nil {
		query += " AND build_id = $2"
		args = append(args, as.build.ID)

		a.Build = as.build
	}

	if as.job != nil {
		if as.build != nil {
			query += " AND job_id = $3"
		} else {
			query += " AND job_id = $2"
		}

		args = append(args, as.job.ID)

		a.Job = as.job
	}

	err := as.Get(a, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return a, errors.Err(err)
}

func (as ArtifactStore) FindByName(name string) (*Artifact, error) {
	a := &Artifact{
		model: model{
			DB: as.DB,
		},
	}

	query := "SELECT * FROM artifacts WHERE name = $1"
	args := []interface{}{name}

	if as.build != nil {
		query += " AND build_id = $2"
		args = append(args, as.build.ID)

		a.Build = as.build
	}

	if as.job != nil {
		if as.build != nil {
			query += " AND job_id = $3"
		} else {
			query += " AND job_id = $2"
		}

		args = append(args, as.job.ID)

		a.Job = as.job
	}

	err := as.Get(a, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return a, errors.Err(err)
}

func (as ArtifactStore) InJobID(ids ...int64) ([]*Artifact, error) {
	aa := make([]*Artifact, 0)

	if len(ids) == 0 {
		return aa, nil
	}

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

func (a *Artifact) Create() error {
	stmt, err := a.Prepare(`
		INSERT INTO artifacts (build_id, job_id, source, name, size, type, md5, sha256)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(a.BuildID, a.JobID, a.Source, a.Name, a.Size, a.Type, a.MD5, a.SHA256)

	return errors.Err(row.Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt))
}

func (a *Artifact) Update() error {
	stmt, err := a.Prepare(`
		UPDATE artifacts
		SET size = $1, type = $2, md5 = $3, sha256 = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(a.Size, a.Type, a.MD5, a.SHA256, a.ID)

	return errors.Err(row.Scan(&a.UpdatedAt))
}
