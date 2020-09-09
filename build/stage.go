package build

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/runner"

	"github.com/andrewpillar/query"

	"github.com/lib/pq"

	"github.com/jmoiron/sqlx"
)

// Stage is the type that represents a stage that is in a build.
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

// StageStore is the type for creating and modifying Stage models in the
// database.
type StageStore struct {
	database.Store

	// Build is the bound Build model. If not nil this will bind the Build
	// model to any Stage models that are created. If not nil this will
	// append a WHERE clause on the build_id column for all SELECT queries
	// performed.
	Build *Build
}

var (
	_ database.Model  = (*Stage)(nil)
	_ database.Binder = (*StageStore)(nil)
	_ database.Loader = (*StageStore)(nil)

	stageTable = "build_stages"
)

// NewStageStore returns a new StageStore for querying the build_stages table.
// Each database passed to this function will be bound to the returned StageStore.
func NewStageStore(db *sqlx.DB, mm ...database.Model) *StageStore {
	s := &StageStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// StageModel is called along with database.ModelSlice to convert the given slice of
// Stage models to a slice of database.Model interfaces.
func StageModel(ss []*Stage) func(int) database.Model {
	return func(i int) database.Model {
		return ss[i]
	}
}

// Bind implements the database.Binder interface. This will only bind the models
// if they are pointers to either Build or Stage.
func (s *Stage) Bind(mm ...database.Model) {
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

// SetPrimary implements the database.Model interface.
func (s *Stage) SetPrimary(id int64) { s.ID = id }

// Primary implements the database.Model interface.
func (s Stage) Primary() (string, int64) { return "id", s.ID }

// IsZero implements the database.Model interface.
func (s *Stage) IsZero() bool {
	return s == nil || s.ID == 0 &&
		s.BuildID == 0 &&
		s.Name == "" &&
		!s.CanFail &&
		s.Status == runner.Status(0) &&
		!s.StartedAt.Valid &&
		!s.FinishedAt.Valid
}

// JSON implements the database.Model interface. This will return a map with the
// current Stage's values under each key. If the Build bound model exists on the
// Stage, then the JSON representation of that model will be in the returned map
// under the build key.
func (s *Stage) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":         s.ID,
		"build_id":   s.BuildID,
		"name":       s.Name,
		"can_fail":   s.CanFail,
		"status":     s.Status.String(),
		"created_at": s.CreatedAt.Format(time.RFC3339),
		"url":        addr + s.Endpoint(),
	}

	if !s.Build.IsZero() {
		json["build"] = s.Build.JSON(addr)
	}
	return json
}

// Endpoint implements the database.Model interface. If the current Stage has a
// nil or zero value Build bound model then an empty string is returned,
// otherwise the full Build endpoint is returned, suffixed with the Stage
// endpoint, for example,
//
//   /b/l.belardo/10/stages/2
func (s *Stage) Endpoint(_ ...string) string {
	if s.Build == nil || s.Build.IsZero() {
		return ""
	}
	return s.Build.Endpoint("stages", strconv.FormatInt(s.ID, 10))
}

// Values implements the database.Model interface. This will return a map with
// the following values, build_id, name, can_fail, status, started_at,
// finished_at.
func (s *Stage) Values() map[string]interface{} {
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

// Bind implements the database.Binder interface. This will only bind the models
// if they are pointers to either Build or Stage.
func (s *StageStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			s.Build = m.(*Build)
		}
	}
}

// Create creates a new Stage model in the database with the name, and whether
// or not it can fail.
func (s *StageStore) Create(name string, canFail bool) (*Stage, error) {
	st := s.New()
	st.Name = name
	st.CanFail = canFail

	err := s.Store.Create(stageTable, st)
	return st, errors.Err(err)
}

// All returns a slice of Stage models, applying each query.Option that is
// given.
func (s StageStore) All(opts ...query.Option) ([]*Stage, error) {
	ss := make([]*Stage, 0)

	opts = append([]query.Option{
		database.Where(s.Build, "build_id"),
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
// given vals. Each database is loaded individually via a call to the given load
// callback. This method calls StageStore.All under the hood, so any bound
// models will impact the models being loaded.
func (s StageStore) Load(key string, vals []interface{}, load database.LoaderFunc) error {
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
