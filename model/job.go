package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type Job struct {
	model

	BuildID    int64          `db:"build_id"`
	StageID    int64          `db:"stage_id"`
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

type JobDependency struct {
	model

	JobID        int64 `db:"job_id"`
	DependencyID int64 `db:"dependency_id"`
}

type JobStore struct {
	*sqlx.DB

	Build *Build
	Stage *Stage
}

type JobDependencyStore struct {
	*sqlx.DB

	Job *Job
}

func (j *Job) ArtifactStore() ArtifactStore {
	return ArtifactStore{
		DB:    j.DB,
		Build: j.Build,
		Job:   j,
	}
}

func (j *Job) JobDependencyStore() JobDependencyStore {
	return JobDependencyStore{
		DB:  j.DB,
		Job: j,
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

func (j *Job) IsZero() bool {
	return j.model.IsZero() &&
           j.BuildID == 0 &&
           j.StageID == 0 &&
           j.Name == "" &&
           j.Commands == "" &&
           j.Status == runner.Status(0) &&
           !j.Output.Valid &&
           j.StartedAt == nil &&
           j.FinishedAt == nil
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

func (jd *JobDependency) Create() error {
	stmt, err := jd.Prepare(`
		INSERT INTO job_dependencies (job_id, dependency_id)
		VALUES ($1, $2)
		RETURNING id
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(jd.JobID, jd.DependencyID)

	return errors.Err(row.Scan(&jd.ID))
}

func (jds JobDependencyStore) New() *JobDependency {
	jd := &JobDependency{
		model: model{
			DB: jds.DB,
		},
	}

	if jds.Job != nil {
		jd.JobID = jds.Job.ID
	}

	return jd
}

func (js JobStore) All() ([]*Job, error) {
	jj := make([]*Job, 0)

	query := "SELECT * FROM jobs"
	args := []interface{}{}

	if js.Build != nil {
		query += " WHERE build_id = $1"
		args = append(args, js.Build.ID)
	}

	if js.Stage != nil {
		if js.Build != nil {
			query += " AND WHERE stage_id = $2"
		} else {
			query += " WHERE stage_id = $1"
		}

		args = append(args, js.Stage.ID)
	}

	err := js.Select(&jj, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, j := range jj {
		j.DB = js.DB
		j.Build = js.Build
		j.Stage = js.Stage
	}

	return jj, errors.Err(err)
}

func (js JobStore) Find(id int64) (*Job, error) {
	j := &Job{
		model: model{
			DB: js.DB,
		},
		Build: js.Build,
		Stage: js.Stage,
	}

	query := "SELECT * FROM jobs WHERE id = $1"
	args := []interface{}{id}

	if js.Build != nil {
		query += " AND build_id = $2"
		args = append(args, js.Build.ID)
	}

	if js.Stage != nil {
		if js.Build != nil {
			query += " AND stage_id = $3"
		} else {
			query += " AND stage_id = $2"
		}

		args = append(args, js.Stage.ID)
	}

	err := js.Get(j, query, args...)

	if err == sql.ErrNoRows {
		err = nil

		j.CreatedAt = nil
		j.UpdatedAt = nil
	}

	return j, errors.Err(err)
}

func (js JobStore) FindByName(name string) (*Job, error) {
	j := &Job{
		model: model{
			DB: js.DB,
		},
		Build: js.Build,
		Stage: js.Stage,
	}

	query := "SELECT * FROM jobs WHERE name = $1"
	args := []interface{}{name}

	if js.Build != nil {
		query += " AND build_id = $2"
		args = append(args, js.Build.ID)
	}

	if js.Stage != nil {
		if js.Build != nil {
			query += " AND stage_id = $3"
		} else {
			query += " AND stage_id = $2"
		}

		args = append(args, js.Stage.ID)
	}

	err := js.Get(j, query, args...)

	if err == sql.ErrNoRows {
		err = nil

		j.CreatedAt = nil
		j.UpdatedAt = nil
	}

	return j, errors.Err(err)
}

func (js JobStore) InStageID(ids ...int64) ([]*Job, error) {
	jj := make([]*Job, 0)

	if len(ids) == 0 {
		return jj, nil
	}

	query, args, err := sqlx.In(`
		SELECT * FROM jobs WHERE stage_id IN (?) ORDER BY created_at ASC
	`, ids)

	if err != nil {
		return jj, errors.Err(err)
	}

	err = js.Select(&jj, js.Rebind(query), args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, j := range jj {
		j.DB = js.DB
		j.Build = js.Build
		j.Stage = js.Stage
	}

	return jj, errors.Err(err)
}

func (j *Job) LoadDependencies() error {
	query := `
		SELECT * FROM jobs
		WHERE id IN (
			SELECT dependency_id FROM job_dependencies
			WHERE job_id = $1
		)
	`

	return errors.Err(j.Select(&j.Dependencies, query, j.ID))
}

func (js JobStore) LoadDependencies(jj []*Job) error {
	if len(jj) == 0 {
		return nil
	}

	ids := make([]int64, len(jj))
	jobs := make(map[int64]*Job)

	for i, j := range jj {
		ids[i] = j.ID
		jobs[j.ID] = j
	}

	query, args, err := sqlx.In("SELECT * FROM job_dependencies WHERE job_id IN (?)")

	if err != nil {
		return errors.Err(err)
	}

	jdd := make([]*JobDependency, 0)

	err = js.Select(&jdd, js.Rebind(query), args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, jd := range jdd {
		job, ok := jobs[jd.JobID]

		if !ok {
			continue
		}

		job.Dependencies = append(job.Dependencies, jobs[jd.DependencyID])
	}

	return errors.Err(err)
}

func (js JobStore) New() *Job {
	j := &Job{
		model: model{
			DB: js.DB,
		},
		Build: js.Build,
		Stage: js.Stage,
	}

	if js.Build != nil {
		j.BuildID = js.Build.ID
	}

	if js.Stage != nil {
		j.StageID = js.Stage.ID
	}

	return j
}

func (js JobStore) NotCompleted() ([]*Job, error) {
	jj := make([]*Job, 0)

	query := "SELECT * FROM jobs WHERE started_at IS NULL AND finished_at IS NULL"
	args := []interface{}{}

	if js.Build != nil {
		query += " AND build_id = $1"
		args = append(args, js.Build.ID)
	}

	err := js.Select(&jj, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, j := range jj {
		j.DB = js.DB
		j.Build = js.Build
		j.Stage = js.Stage
	}

	return jj, errors.Err(err)
}
