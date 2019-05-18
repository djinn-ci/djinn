package model

import (
	"database/sql"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type Job struct {
	model

	BuildID    int64          `db:"build_id"`
	StageID    int64          `db:"stage_id"`
	ParentID   sql.NullInt64  `db:"parent_id"`
	Name       string         `db:"name"`
	Commands   string         `db:"commands"`
	Status     runner.Status  `db:"status"`
	Output     sql.NullString `db:"output"`
	StartedAt  *pq.NullTime   `db:"started_at"`
	FinishedAt *pq.NullTime   `db:"finished_at"`

	Build        *Build
	Stage        *Stage
	Artifacts    []*Artifact
	Dependencies []*Job
}

type JobStore struct {
	*sqlx.DB

	build *Build
	stage *Stage
}

func (js JobStore) New() *Job {
	j := &Job{
		model: model{
			DB: js.DB,
		},
	}

	if js.build != nil {
		j.BuildID = js.build.ID
		j.Build = js.build
	}

	if js.stage != nil {
		j.StageID = js.stage.ID
		j.Stage = js.stage
	}

	return j
}

func (js JobStore) All() ([]*Job, error) {
	jj := make([]*Job, 0)

	query := "SELECT * FROM jobs"
	args := []interface{}{}

	if js.build != nil {
		query += " WHERE build_id = $1"
		args = append(args, js.build.ID)
	}

	if js.stage != nil {
		if js.build != nil {
			query += " AND WHERE stage_id = $2"
		} else {
			query += " WHERE stage_id = $1"
		}

		args = append(args, js.stage.ID)
	}

	err := js.Select(&jj, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, j := range jj {
		j.DB = js.DB
		j.Build = js.build
		j.Stage = js.stage
	}

	return jj, errors.Err(err)
}

func (js JobStore) Find(id int64) (*Job, error) {
	j := &Job{
		model: model{
			DB: js.DB,
		},
	}

	query := "SELECT * FROM jobs WHERE id = $1"
	args := []interface{}{id}

	if js.build != nil {
		query += " AND build_id = $2"
		args = append(args, js.build.ID)

		j.Build = js.build
	}

	if js.stage != nil {
		if js.build != nil {
			query += " AND stage_id = $3"
		} else {
			query += " AND stage_id = $2"
		}

		args = append(args, js.stage.ID)

		j.Stage = js.stage
	}

	err := js.Get(j, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return j, errors.Err(err)
}

func (js JobStore) InStageID(ids ...int64) ([]*Job, error) {
	jj := make([]*Job, 0)

	if len(ids) == 0 {
		return jj, nil
	}

	query, args, err := sqlx.In("SELECT * FROM jobs WHERE stage_id IN (?)", ids)

	if err != nil {
		return jj, errors.Err(err)
	}

	err = js.Select(&jj, js.Rebind(query), args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, j := range jj {
		j.DB = js.DB
	}

	return jj, errors.Err(err)
}

func (js JobStore) InParentID(ids ...int64) ([]*Job, error) {
	jj := make([]*Job, 0)

	if len(ids) == 0 {
		return jj, nil
	}

	query, args, err := sqlx.In("SELECT * FROM jobs WHERE parent_id IN (?)", ids)

	if err != nil {
		return jj, errors.Err(err)
	}

	err = js.Select(&jj, js.Rebind(query), args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, j := range jj {
		j.DB = js.DB
	}

	return jj, errors.Err(err)
}

func (j *Job) ArtifactStore() ArtifactStore {
	return ArtifactStore{
		DB:    j.DB,
		build: j.Build,
		job:   j,
	}
}

func (j *Job) Create() error {
	stmt, err := j.Prepare(`
		INSERT INTO jobs (build_id, stage_id, name, commands)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(j.BuildID, j.StageID, j.Name, j.Commands)

	return errors.Err(row.Scan(&j.ID, &j.CreatedAt, &j.UpdatedAt))
}

func (j *Job) Update() error {
	stmt, err := j.Prepare(`
		UPDATE jobs
		SET output = $1, status = $2, started_at = $3, finished_at = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(j.Output, j.Status, j.StartedAt, j.FinishedAt, j.ID)

	return errors.Err(row.Scan(&j.UpdatedAt))
}

func (j Job) Job() *runner.Job {
	artifacts := runner.NewPassthrough()

	for _, a := range j.Artifacts {
		artifacts[a.Source] = a.Name
	}

	depends := make([]string, len(j.Dependencies), len(j.Dependencies))

	for i, d := range j.Dependencies {
		depends[i] = d.Name
	}

	return &runner.Job{
		Stage:     j.Stage.Name,
		Name:      j.Name,
		Commands:  strings.Split(j.Commands, "\n"),
		Depends:   depends,
		Artifacts: artifacts,
	}
}

func (j *Job) IsZero() bool {
	return j.ID == 0 &&
           j.BuildID == 0 &&
           j.StageID == 0 &&
           !j.ParentID.Valid &&
           j.Name == "" &&
           j.Commands == "" &&
           j.Status == runner.Status(0) &&
           !j.Output.Valid &&
           j.StartedAt == nil &&
           j.FinishedAt == nil &&
           j.CreatedAt == nil &&
           j.UpdatedAt == nil
}
