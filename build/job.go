package build

import (
	"database/sql"
	"fmt"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/runner"

	"github.com/andrewpillar/query"

	"github.com/lib/pq"

	"github.com/jmoiron/sqlx"
)

type Job struct {
	ID         int64          `db:"id"`
	BuildID    int64          `db:"build_id"`
	StageID    int64          `db:"stage_id"`
	Name       string         `db:"name"`
	Commands   string         `db:"commands"`
	Status     runner.Status  `db:"status"`
	Output     sql.NullString `db:"output"`
	StartedAt  pq.NullTime    `db:"started_at"`
	FinishedAt pq.NullTime    `db:"finished_at"`

	Build        *Build      `db:"-"`
	Stage        *Stage      `db:"-"`
	Artifacts    []*Artifact `db:"-"`
	Dependencies []*Job      `db:"-"`
}

type JobStore struct {
	model.Store

	Build *Build
	Stage *Stage
}

var (
	_ model.Model  = (*Job)(nil)
	_ model.Binder = (*JobStore)(nil)
	_ model.Loader = (*JobStore)(nil)

	jobTable = "build_jobs"
)

func NewJobStore(db *sqlx.DB, mm ...model.Model) JobStore {
	s := JobStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func JobModel(jj []*Job) func(int) model.Model {
	return func(i int) model.Model {
		return jj[i]
	}
}

func (j *Job) Bind(mm ...model.Model) {
	if j == nil {
		return
	}

	for _, m := range mm {
		switch m.(type) {
		case *Build:
			j.Build = m.(*Build)
		case *Stage:
			j.Stage = m.(*Stage)
		case *Artifact:
			j.Artifacts = append(j.Artifacts, m.(*Artifact))
		case *Job:
			j.Dependencies = append(j.Dependencies, m.(*Job))
		}
	}
}

func (*Job) Kind() string { return "build_job" }

func (j *Job) SetPrimary(id int64) {
	if j == nil {
		return
	}
	j.ID = id
}

func (j *Job) Primary() (string, int64) {
	if j == nil {
		return "id", 0
	}
	return "id", j.ID
}

func (j *Job) IsZero() bool {
	return j == nil || j.ID == 0 &&
		j.BuildID == 0 &&
		j.StageID == 0 &&
		j.Name == "" &&
		j.Commands == "" &&
		j.Status == runner.Status(0) &&
		!j.Output.Valid &&
		!j.StartedAt.Valid &&
		!j.FinishedAt.Valid
}

func (j Job) Endpoint(uri ...string) string {
	if j.Build == nil || j.Build.IsZero() {
		return ""
	}

	uri = append([]string{"jobs", fmt.Sprintf("%v", j.ID)}, uri...)
	return j.Build.Endpoint(uri...)
}

func (j Job) Values() map[string]interface{} {
	return map[string]interface{}{
		"build_id":    j.BuildID,
		"stage_id":    j.StageID,
		"name":        j.Name,
		"commands":    j.Commands,
		"status":      j.Status,
		"output":      j.Output,
		"started_at":  j.StartedAt,
		"finished_at": j.FinishedAt,
	}
}

func (s JobStore) New() *Job {
	j := &Job{
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

func (s *JobStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			s.Build = m.(*Build)
		case *Stage:
			s.Stage = m.(*Stage)
		}
	}
}

func (s JobStore) Create(jj ...*Job) error {
	models := model.Slice(len(jj), JobModel(jj))
	return errors.Err(s.Store.Create(jobTable, models...))
}

func (s JobStore) Update(jj ...*Job) error {
	models := model.Slice(len(jj), JobModel(jj))
	return errors.Err(s.Store.Update(jobTable, models...))
}

func (s JobStore) Get(opts ...query.Option) (*Job, error) {
	j := &Job{
		Build: s.Build,
		Stage: s.Stage,
	}

	opts = append([]query.Option{
		model.Where(s.Build, "build_id"),
		model.Where(s.Stage, "stage_id"),
	}, opts...)

	err := s.Store.Get(j, jobTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return j, errors.Err(err)
}

func (s JobStore) All(opts ...query.Option) ([]*Job, error) {
	jj := make([]*Job, 0)

	opts = append([]query.Option{
		model.Where(s.Build, "build_id"),
		model.Where(s.Stage, "stage_id"),
	}, opts...)

	err := s.Store.All(&jj, jobTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, j := range jj {
		j.Build = s.Build
		j.Stage = s.Stage
	}
	return jj, errors.Err(err)
}

func (s JobStore) Load(key string, vals []interface{}, load model.LoaderFunc) error {
	jj, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, j := range jj {
			load(i, j)
		}
	}
	return nil
}
