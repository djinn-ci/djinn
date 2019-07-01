package model

import (
	"database/sql"
	"fmt"
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
	Name       string         `db:"name"`
	Commands   string         `db:"commands"`
	Status     runner.Status  `db:"status"`
	Output     sql.NullString `db:"output"`
	StartedAt  pq.NullTime    `db:"started_at"`
	FinishedAt pq.NullTime    `db:"finished_at"`

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
	q := Insert(
		Table("jobs"),
		Columns("build_id", "stage_id", "name", "commands"),
		Values(j.BuildID, j.StageID, j.Name, j.Commands),
		Returning("id", "created_at", "updated_at"),
	)

	stmt, err := j.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

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
           !j.StartedAt.Valid &&
           !j.FinishedAt.Valid
}

func (j *Job) LoadArtifacts() error {
	var err error

	j.Artifacts, err = j.ArtifactStore().All()

	return errors.Err(err)
}

func (j *Job) LoadBuild() error {
	var err error

	builds := BuildStore{
		DB: j.DB,
	}

	j.Build, err = builds.Find(j.BuildID)

	return errors.Err(err)
}

func (j *Job) LoadDependencies() error {
	q := Select(
		Columns("*"),
		Table("jobs"),
		WhereInQuery("id", Select(
				Columns("dependency_id"),
				Table("job_dependencies"),
				WhereEq("job_id", j.ID),
			),
		),
	)

	return errors.Err(j.Select(&j.Dependencies, q.Build(), q.Args()...))
}

func (j *Job) LoadStage() error {
	var err error

	stages := StageStore{
		DB: j.DB,
	}

	j.Stage, err = stages.Find(j.StageID)

	return errors.Err(err)
}

func (j Job) UIEndpoint(uri ...string) string {
	endpoint :=  fmt.Sprintf("/builds/%v/jobs/%v", j.BuildID, j.ID)

	if len(uri) > 0 {
		endpoint = fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}

	return endpoint
}

func (j *Job) Update() error {
	q := Update(
		Table("jobs"),
		Set("output", j.Output),
		Set("status", j.Status),
		Set("started_at", j.StartedAt),
		Set("finished_at", j.FinishedAt),
		SetRaw("updated_at", "NOW()"),
		WhereEq("id", j.ID),
		Returning("updated_at"),
	)

	stmt, err := j.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&j.UpdatedAt))
}

func (jd *JobDependency) Create() error {
	q := Insert(
		Table("job_dependencies"),
		Columns("job_id", "dependency_id"),
		Values(jd.JobID, jd.DependencyID),
		Returning("id"),
	)

	stmt, err := jd.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

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

func (js JobStore) All(opts ...Option) ([]*Job, error) {
	jj := make([]*Job, 0)

	opts = append([]Option{Columns("*")}, opts...)

	q := Select(append(opts, ForBuild(js.Build), ForStage(js.Stage), Table("jobs"))...)

	err := js.Select(&jj, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, j := range jj {
		j.DB = js.DB
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

	q := Select(
		Columns("*"),
		Table("jobs"),
		WhereEq("id", id),
		ForBuild(js.Build),
		ForStage(js.Stage),
	)

	err := js.Get(j, q.Build(), q.Args()...)

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

	q := Select(
		Columns("*"),
		Table("jobs"),
		WhereEq("name", name),
		ForBuild(js.Build),
		ForStage(js.Stage),
	)

	err := js.Get(j, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil

		j.CreatedAt = nil
		j.UpdatedAt = nil
	}

	return j, errors.Err(err)
}

func (js JobStore) LoadArtifacts(jj []*Job) error {
	if len(jj) == 0 {
		return nil
	}

	ids := make([]interface{}, len(jj))

	for i, j := range jj {
		ids[i] = j.ID
	}

	artifacts := ArtifactStore{
		DB: js.DB,
	}

	aa, err := artifacts.All(WhereIn("job_id", ids...))

	if err != nil {
		return errors.Err(err)
	}

	for _, j := range jj {
		for _, a := range aa {
			if j.ID == a.JobID {
				j.Artifacts = append(j.Artifacts, a)
			}
		}
	}

	return nil
}

func (js JobStore) LoadDependencies(jj []*Job) error {
	if len(jj) == 0 {
		return nil
	}

	ids := make([]interface{}, len(jj))
	jobs := make(map[int64]*Job)

	for i, j := range jj {
		ids[i] = j.ID
		jobs[j.ID] = j
	}

	dependencies := JobDependencyStore{
		DB: js.DB,
	}

	jdd, err := dependencies.All(WhereIn("job_id", ids...))

	if err != nil {
		return errors.Err(err)
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

func (jds JobDependencyStore) All(opts ...Option) ([]*JobDependency, error) {
	jdd := make([]*JobDependency, 0)

	opts = append([]Option{Columns("*")}, opts...)

	q := Select(append(opts, ForJob(jds.Job), Table("job_dependencies"))...)

	err := jds.Select(&jdd, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, d := range jdd {
		d.DB = jds.DB
	}

	return jdd, errors.Err(err)
}
