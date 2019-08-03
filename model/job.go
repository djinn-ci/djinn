package model

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"
	"github.com/andrewpillar/thrall/runner"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type Job struct {
	Model

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
	Model

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
	q := query.Insert(
		query.Table("jobs"),
		query.Columns("build_id", "stage_id", "name", "commands"),
		query.Values(j.BuildID, j.StageID, j.Name, j.Commands),
		query.Returning("id", "created_at", "updated_at"),
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
	return j.Model.IsZero() &&
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
	q := query.Select(
		query.Columns("*"),
		query.Table("jobs"),
		query.WhereInQuery("id", query.Select(
				query.Columns("dependency_id"),
				query.Table("job_dependencies"),
				query.WhereEq("job_id", j.ID),
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
	q := query.Update(
		query.Table("jobs"),
		query.Set("output", j.Output),
		query.Set("status", j.Status),
		query.Set("started_at", j.StartedAt),
		query.Set("finished_at", j.FinishedAt),
		query.SetRaw("updated_at", "NOW()"),
		query.WhereEq("id", j.ID),
		query.Returning("updated_at"),
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
	q := query.Insert(
		query.Table("job_dependencies"),
		query.Columns("job_id", "dependency_id"),
		query.Values(jd.JobID, jd.DependencyID),
		query.Returning("id"),
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
		Model: Model{
			DB: jds.DB,
		},
	}

	if jds.Job != nil {
		jd.JobID = jds.Job.ID
	}

	return jd
}

func (js JobStore) All(opts ...query.Option) ([]*Job, error) {
	jj := make([]*Job, 0)

	opts = append([]query.Option{query.Columns("*")}, opts...)
	opts = append(opts, ForBuild(js.Build), ForStage(js.Stage), query.Table("jobs"))

	q := query.Select(opts...)

	err := js.Select(&jj, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, j := range jj {
		j.DB = js.DB
	}

	return jj, errors.Err(err)
}

func (js JobStore) findBy(col string, val interface{}) (*Job, error) {
	j := &Job{
		Model: Model{
			DB: js.DB,
		},
		Build: js.Build,
		Stage: js.Stage,
	}

	q := query.Select(
		query.Columns("*"),
		query.Table("jobs"),
		query.WhereEq(col, val),
		ForBuild(js.Build),
		ForStage(js.Stage),
	)

	err := js.Get(j, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return j, errors.Err(err)
}

func (js JobStore) Find(id int64) (*Job, error) {
	j, err := js.findBy("id", id)

	return j, errors.Err(err)
}

func (js JobStore) FindByName(name string) (*Job, error) {
	j, err := js.findBy("name", name)

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

	aa, err := artifacts.All(query.WhereIn("job_id", ids...))

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

	jdd, err := dependencies.All(query.WhereIn("job_id", ids...))

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
		Model: Model{
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

func (js JobStore) Show(id int64) (*Job, error) {
	j, err := js.Find(id)

	if err != nil {
		return j, errors.Err(err)
	}

	if err := j.LoadStage(); err != nil {
		return j, errors.Err(err)
	}

	if err := j.LoadDependencies(); err != nil {
		return j, errors.Err(err)
	}

	return j, errors.Err(j.LoadArtifacts())
}

func (jds JobDependencyStore) All(opts ...query.Option) ([]*JobDependency, error) {
	jdd := make([]*JobDependency, 0)

	opts = append([]query.Option{query.Columns("*")}, opts...)
	opts = append(opts, ForJob(jds.Job), query.Table("job_dependencies"))

	q := query.Select(opts...)

	err := jds.Select(&jdd, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, d := range jdd {
		d.DB = jds.DB
	}

	return jdd, errors.Err(err)
}
