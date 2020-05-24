package build

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/runner"

	"github.com/andrewpillar/query"

	"github.com/lib/pq"

	"github.com/jmoiron/sqlx"
)

type Stage struct {
	ID         int64         `db:"id"`
	BuildID    int64         `db:"build_id"`
	Name       string        `db:"name"`
	CanFail    bool          `db:"can_fail"`
	Status     runner.Status `db:"status"`
	CreatedAt  time.Time     `db:"created_at"`
	StartedAt  pq.NullTime   `db:"started_at"`
	FinishedAt pq.NullTime   `db:"finished_at"`

	Build *Build `db:"-"`
	Jobs  []*Job `db:"-"`
}

type StageStore struct {
	model.Store

	Build *Build
}

var (
	_ model.Model  = (*Stage)(nil)
	_ model.Binder = (*StageStore)(nil)
	_ model.Loader = (*StageStore)(nil)

	stageTable = "build_stages"
)

// NewStageStore returns a new StageStore for querying the build_stages table.
// Each model passed to this function will be bound to the returned StageStore.
func NewStageStore(db *sqlx.DB, mm ...model.Model) *StageStore {
	s := &StageStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// StageModel is called along with model.Slice to convert the given slice of
// Stage models to a slice of model.Model interfaces.
func StageModel(ss []*Stage) func(int) model.Model {
	return func(i int) model.Model {
		return ss[i]
	}
}

// Bind the given models to the current Stage. This will only bind the model if
// they are one of the following,
//
// - *Build
// - *Job
func (s *Stage) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			s.Build = m.(*Build)
		case *Job:
			j := m.(*Job)
			j.Build = s.Build
			s.Jobs = append(s.Jobs, j)
		}
	}
}

func (s *Stage) SetPrimary(id int64) {
	s.ID = id
}

func (s Stage) Primary() (string, int64) {
	return "id", s.ID
}

func (s *Stage) IsZero() bool {
	return s == nil || s.ID == 0 &&
		s.BuildID == 0 &&
		s.Name == "" &&
		!s.CanFail &&
		s.Status == runner.Status(0) &&
		!s.StartedAt.Valid &&
		!s.FinishedAt.Valid
}

func (s *Stage) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":          s.ID,
		"build_id":    s.BuildID,
		"name":        s.Name,
		"can_fail":    s.CanFail,
		"status":      s.Status.String(),
		"created_at":  s.CreatedAt.Format(time.RFC3339),
		"url":         addr + s.Endpoint(),
	}

	if !s.Build.IsZero() {
		json["build"] = s.Build.JSON(addr)
	}
	return json
}

func (s *Stage) Endpoint(_ ...string) string {
	if s.Build == nil || s.Build.IsZero() {
		return ""
	}
	return s.Build.Endpoint("stages", strconv.FormatInt(s.ID, 10))
}

func (s *Stage) Values() map[string]interface{} {
	if s == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"build_id":    s.BuildID,
		"name":        s.Name,
		"can_fail":    s.CanFail,
		"status":      s.Status,
		"started_at":  s.StartedAt,
		"finished_at": s.FinishedAt,
	}
}

// Stage returns the underlying runner.Stage of the current Stage.
func (s Stage) Stage() *runner.Stage {
	return &runner.Stage{
		Name:    s.Name,
		CanFail: s.CanFail,
	}
}

// New returns a new Stage binding any non-nil models to it from the current
// StageStore.
func (s StageStore) New() *Stage {
	st := &Stage{
		Build: s.Build,
	}

	if s.Build != nil {
		st.BuildID = s.Build.ID
	}
	return st
}

// Bind the given models to the current StageStore. This will only bind the
// model if they are one of the following,
//
// - *Build
func (s *StageStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			s.Build = m.(*Build)
		}
	}
}

// Create inserts the given Stage models into the build_stages table.
func (s StageStore) Create(ss ...*Stage) error {
	models := model.Slice(len(ss), StageModel(ss))
	return errors.Err(s.Store.Create(stageTable, models...))
}

// All returns a slice of Stage models, applying each query.Option that is
// given. The model.Where option is used on the Build bound model to limit the
// query to those relations.
func (s StageStore) All(opts ...query.Option) ([]*Stage, error) {
	ss := make([]*Stage, 0)

	opts = append([]query.Option{
		model.Where(s.Build, "build_id"),
	}, opts...)

	err := s.Store.All(&ss, stageTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, st := range ss {
		st.Build = s.Build
	}
	return ss, errors.Err(err)
}

// Load loads in a slice of Stage models where the given key is in the list of
// given vals. Each model is loaded individually via a call to the given load
// callback. This method calls StageStore.All under the hood, so any bound
// models will impact the models being loaded.
func (s StageStore) Load(key string, vals []interface{}, load model.LoaderFunc) error {
	ss, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, s := range ss {
			load(i, s)
		}
	}
	return nil
}
