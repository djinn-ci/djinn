package model

import (
	"database/sql"
	"strings"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type Job struct {
	model

	BuildID    int64          `db:"build_id"`
	StageID    int64          `db:"stage_id"`
	Parent     sql.NullInt64  `db:"parent"`
	Name       string         `db:"name"`
	Commands   string         `db:"commands"`
	Status     Status         `db:"status"`
	Output     sql.NullString `db:"output"`
	StartedAt  *pq.NullTime   `db:"started_at"`
	FinishedAt *pq.NullTime   `db:"finished_at"`

	Build        *Build
	Stage        *Stage
	Artifacts    []*Artifact
	Dependencies []*Job
}

type JobStore struct {
	*Store

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

func (js JobStore) InParentID(ids ...int64) ([]*Job, error) {
	jj := make([]*Job, 0)

	if len(ids) == 0 {
		return jj, nil
	}

	query, args, err := sqlx.In("SELECT * FROM namespaces WHERE parent_id IN (?)", ids)

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

func (js JobStore) LoadRelations(jj []*Job) error {
	if len(jj) == 0 {
		return nil
	}

	stageIds := make([]int64, len(jj), len(jj))
	buildIds := make([]int64, len(jj), len(jj))
	jobIds := make([]int64, len(jj), len(jj))

	for i, j := range jj {
		stageIds[i] = j.StageID
		buildIds[i] = j.BuildID
		jobIds[i] = j.ID
	}

	stages := StageStore{
		Store: &Store{
			DB: js.DB,
		},
	}

	ss, err := stages.In(stageIds...)

	if err != nil {
		return errors.Err(err)
	}

	builds := BuildStore{
		Store: &Store{
			DB: js.DB,
		},
	}

	bb, err := builds.In(buildIds...)

	if err != nil {
		return errors.Err(err)
	}

	jobs := JobStore{
		Store: &Store{
			DB: js.DB,
		},
	}

	deps, err := jobs.InParentID(jobIds...)

	if err != nil {
		return errors.Err(err)
	}

	artifacts := ArtifactStore{
		Store: &Store{
			DB: js.DB,
		},
	}

	aa, err := artifacts.InJobID(jobIds...)

	if err != nil {
		return errors.Err(err)
	}

	for _, j := range jj {
		for _, s := range ss {
			if j.StageID == s.ID {
				j.Stage = s
			}
		}

		for _, b := range bb {
			if j.BuildID == b.ID {
				j.Build = b
			}
		}

		for _, a := range aa {
			if a.JobID == j.ID {
				j.Artifacts = append(j.Artifacts, a)
			}
		}

		for _, d := range deps {
			if d.Parent.Int64 == j.ID {
				j.Dependencies = append(j.Dependencies, d)
			}
		}
	}

	return nil
}

func (j *Job) ArtifactStore() ArtifactStore {
	return ArtifactStore{
		Store: &Store{
			DB: j.DB,
		},
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
	artifacts := make([]config.Passthrough, len(j.Artifacts), len(j.Artifacts))

	for i, a := range j.Artifacts {
		artifacts[i] = config.Passthrough{
			Source:      a.Source,
			Destination: a.Name,
		}
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
           !j.Parent.Valid &&
           j.Name == "" &&
           j.Commands == "" &&
           j.Status == Status(0) &&
           !j.Output.Valid &&
           j.StartedAt == nil &&
           j.FinishedAt == nil &&
           j.CreatedAt == nil &&
           j.UpdatedAt == nil
}

func (j *Job) LoadRelations() error {
	var err error

	if j.Build == nil {
		builds := BuildStore{
			Store: &Store{
				DB: j.DB,
			},
		}

		j.Build, err = builds.Find(j.BuildID)

		if err != nil {
			return errors.Err(err)
		}
	}

	if j.Stage == nil {
		stages := StageStore{
			Store: &Store{
				DB: j.DB,
			},
		}

		j.Stage, err = stages.Find(j.StageID)

		if err != nil {
			return errors.Err(err)
		}
	}

	return nil
}
