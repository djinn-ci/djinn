package build

import (
	"database/sql"

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

func NewStageStore(db *sqlx.DB, mm ...model.Model) StageStore {
	s := StageStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func StageModel(ss []*Stage) func(int) model.Model {
	return func(i int) model.Model {
		return ss[i]
	}
}

func (s *Stage) Bind(mm ...model.Model) {
	if s == nil {
		return
	}

	for _, m := range mm {
		switch m.(type) {
		case *Build:
			s.Build = m.(*Build)
		case *Job:
			s.Jobs = append(s.Jobs, m.(*Job))
		}
	}
}

func (*Stage) Kind() string { return "build_stage" }

func (s *Stage) SetPrimary(id int64) {
	if s == nil {
		return
	}
	s.ID = id
}

func (s *Stage) Primary() (string, int64) {
	if s == nil {
		return "id", 0
	}
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

func (*Stage) Endpoint(_ ...string) string { return "" }

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

func (s StageStore) New() *Stage {
	st := &Stage{
		Build: s.Build,
	}

	if s.Build != nil {
		st.BuildID = s.Build.ID
	}
	return st
}

func (s *StageStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			s.Build = m.(*Build)
		}
	}
}

func (s StageStore) Create(ss ...*Stage) error {
	models := model.Slice(len(ss), StageModel(ss))
	return errors.Err(s.Store.Create(stageTable, models...))
}

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
