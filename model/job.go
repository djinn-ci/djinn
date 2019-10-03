package model

import (
	"database/sql"
	"fmt"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"github.com/andrewpillar/query"

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

type JobStore struct {
	Store

	Build *Build
	Stage *Stage
}

func jobToInterface(jj ...*Job) func(i int) Interface {
	return func(i int) Interface {
		return jj[i]
	}
}

func (j *Job) ArtifactStore() ArtifactStore {
	return ArtifactStore{
		Store: Store{
			DB: j.DB,
		},
		Build: j.Build,
		Job:   j,
	}
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
		Store: Store{
			DB: j.DB,
		},
	}

	j.Build, err = builds.Find(j.BuildID)

	return errors.Err(err)
}

func (j *Job) LoadStage() error {
	var err error

	stages := StageStore{
		Store: Store{
			DB: j.DB,
		},
	}

	j.Stage, err = stages.Find(j.StageID)

	return errors.Err(err)
}

func (j Job) UIEndpoint(uri ...string) string {
	if j.Build == nil || j.Build.IsZero() {
		return ""
	}

	uri = append([]string{"jobs", fmt.Sprintf("%v", j.ID)}, uri...)

	return j.Build.UIEndpoint(uri...)
}

func (j Job) Values() map[string]interface{} {
	return map[string]interface{}{
		"build_id":    j.BuildID,
		"stage_id":    j.StageID,
		"name":        j.Name,
		"commands":    j.Commands,
		"output":      j.Output,
		"status":      j.Status,
		"started_at":  j.StartedAt,
		"finished_at": j.FinishedAt,
	}
}

func (s JobStore) Create(jj ...*Job) error {
	models := interfaceSlice(len(jj), jobToInterface(jj...))

	return errors.Err(s.Store.Create(JobTable, models...))
}

func (s JobStore) All(opts ...query.Option) ([]*Job, error) {
	jj := make([]*Job, 0)

	opts = append(opts, ForBuild(s.Build), ForStage(s.Stage))

	err := s.Store.All(&jj, JobTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, j := range jj {
		j.DB = s.DB
		j.Build = s.Build
	}

	return jj, errors.Err(err)
}

func (s JobStore) findBy(col string, val interface{}) (*Job, error) {
	j := &Job{
		Model: Model{
			DB: s.DB,
		},
		Build: s.Build,
		Stage: s.Stage,
	}

	err := s.FindBy(j, JobTable, col, val)

	return j, errors.Err(err)
}

func (s JobStore) Find(id int64) (*Job, error) {
	j, err := s.findBy("id", id)

	return j, errors.Err(err)
}

func (s JobStore) FindByName(name string) (*Job, error) {
	j, err := s.findBy("name", name)

	return j, errors.Err(err)
}

func (s JobStore) LoadArtifacts(jj []*Job) error {
	if len(jj) == 0 {
		return nil
	}

	ids := make([]interface{}, len(jj))

	for i, j := range jj {
		ids[i] = j.ID
	}

	artifacts := ArtifactStore{
		Store: s.Store,
	}

	aa, err := artifacts.All(query.Where("job_id", "IN", ids...))

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

func (s JobStore) New() *Job {
	j := &Job{
		Model: Model{
			DB: s.DB,
		},
		Build: s.Build,
		Stage: s.Stage,
	}

	if s.Build != nil {
		j.BuildID = s.Build.ID
	}

	if s.Stage != nil {
		j.StageID = s.Stage.ID
	}

	return j
}

func (s JobStore) Update(jj ...*Job) error {
	models := interfaceSlice(len(jj), jobToInterface(jj...))

	return errors.Err(s.Store.Update(JobTable, models...))
}
