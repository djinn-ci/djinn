package build

import (
	"database/sql"
	"fmt"
	"io"
	"strings"
	"time"

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
	CreatedAt  time.Time      `db:"created_at"`
	StartedAt  pq.NullTime    `db:"started_at"`
	FinishedAt pq.NullTime    `db:"finished_at"`

	Build        *Build      `db:"-"`
	Stage        *Stage      `db:"-"`
	Artifacts    []*Artifact `db:"-"`
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

// NewJobStore returns a new JobStore for querying the build_jobs table. Each
// model passed to this function will be bound to the returned JobStore.
func NewJobStore(db *sqlx.DB, mm ...model.Model) *JobStore {
	s := &JobStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// JobModel is called along with model.Slice to convert the given slice of
// Job models to a slice of model.Model interfaces.
func JobModel(jj []*Job) func(int) model.Model {
	return func(i int) model.Model {
		return jj[i]
	}
}

// Bind the given models to the current Job. This will only bind the model if
// they are one of the following,
//
// - *Build
// - *Stage
// - *Artifact
func (j *Job) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			j.Build = m.(*Build)
		case *Stage:
			j.Stage = m.(*Stage)
		case *Artifact:
			j.Artifacts = append(j.Artifacts, m.(*Artifact))
		}
	}
}

func (j *Job) SetPrimary(id int64) {
	j.ID = id
}

func (j *Job) Primary() (string, int64) {
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

// Endpoint returns the endpoint for the current Job. If nil, or if missing
// a bound Build model, then an empty string is returned.
func (j *Job) Endpoint(uri ...string) string {
	if j.Build == nil || j.Build.IsZero() {
		return ""
	}

	uri = append([]string{"jobs", fmt.Sprintf("%v", j.ID)}, uri...)
	return j.Build.Endpoint(uri...)
}

func (j *Job) Values() map[string]interface{} {
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

// Job returns the underlying runner.Job of the current Job. This can then be
// passed to a runner.Driver for execution. It is expected for the Job's
// Artifact slice to be already loaded onto the current model.
func (j *Job) Job(w io.Writer) *runner.Job {
	artifacts := runner.Passthrough{}

	for _, a := range j.Artifacts {
		artifacts.Set(a.Source, a.Name)
	}

	return &runner.Job{
		Writer:    w,
		Name:      j.Name,
		Commands:  strings.Split(j.Commands, "\n"),
		Artifacts: artifacts,
	}
}

// New returns a new Job binding any non-nil models to it from the current
// JobStore.
func (s *JobStore) New() *Job {
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

// Bind the given models to the current JobStore. This will only bind the model
// if they are one of the following,
//
// - *Build
// - *Stage
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

// Create inserts the given Job models into the build_jobs table.
func (s *JobStore) Create(jj ...*Job) error {
	models := model.Slice(len(jj), JobModel(jj))
	return errors.Err(s.Store.Create(jobTable, models...))
}

// Update updates the given Job models in the build_jobs table.
func (s *JobStore) Update(jj ...*Job) error {
	models := model.Slice(len(jj), JobModel(jj))
	return errors.Err(s.Store.Update(jobTable, models...))
}

// Get returns a single Job model, applying each query.Option that is given.
// The model.Where option is used on the Build and Stage bound models to limit
// the query to those relations.
func (s *JobStore) Get(opts ...query.Option) (*Job, error) {
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

// All returns a slice of Job models, applying each query.Option that is given.
// The model.Where option is used on the Build and Stage bound models to limit
// the query to those relations.
func (s *JobStore) All(opts ...query.Option) ([]*Job, error) {
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

// Load loads in a slice of Job models where the given key is in the list of
// given vals. Each model is loaded individually via a call to the given load
// callback. This method calls JobStore.All under the hood, so any bound models
// will impact the models being loaded.
func (s *JobStore) Load(key string, vals []interface{}, load model.LoaderFunc) error {
	jj, err := s.All(query.Where(key, "IN", vals...), query.OrderAsc("created_at"))

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
