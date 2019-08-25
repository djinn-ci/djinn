package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"
	"github.com/andrewpillar/thrall/runner"

	"github.com/lib/pq"
)

type Stage struct {
	Model

	BuildID    int64         `db:"build_id"`
	Name       string        `db:"name"`
	CanFail    bool          `db:"can_fail"`
	Status     runner.Status `db:"status"`
	StartedAt  pq.NullTime   `db:"started_at"`
	FinishedAt pq.NullTime   `db:"finished_at"`

	Build *Build
	Jobs  []*Job
}

type StageStore struct {
	Store

	Build *Build
}

func stageToInterface(ss []*Stage) func(i int) Interface {
	return func(i int) Interface {
		return ss[i]
	}
}

func (s *Stage) JobStore() JobStore {
	return JobStore{
		Store: Store{
			DB: s.DB,
		},
		Build: s.Build,
		Stage: s,
	}
}

func (s Stage) Values() map[string]interface{} {
	return map[string]interface{}{
		"build_id":    s.BuildID,
		"name":        s.Name,
		"can_fail":    s.CanFail,
		"status":      s.Status,
		"started_at":  s.StartedAt,
		"finished_at": s.FinishedAt,
	}
}

func (s StageStore) All(opts ...query.Option) ([]*Stage, error) {
	ss := make([]*Stage, 0)

	opts = append(opts, ForBuild(s.Build))

	err := s.Store.All(&ss, StageTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, st := range ss {
		st.DB = s.DB

		if s.Build != nil {
			st.Build = s.Build
		}
	}

	return ss, errors.Err(err)
}

func (s StageStore) Create(ss ...*Stage) error {
	models := interfaceSlice(len(ss), stageToInterface(ss))

	return errors.Err(s.Store.Create(StageTable, models...))
}

func (s StageStore) findBy(col string, val interface{}) (*Stage, error) {
	st := &Stage{
		Model: Model{
			DB: s.DB,
		},
	}

	err := s.FindBy(st, StageTable, col, val)

	if err == sql.ErrNoRows {
		err = nil
	}

	return st, errors.Err(err)
}

func (s StageStore) Find(id int64) (*Stage, error) {
	st, err := s.findBy("id", id)

	return st, errors.Err(err)
}

func (s StageStore) FindByName(name string) (*Stage, error) {
	st, err := s.findBy("name", name)

	return st, errors.Err(err)
}

func (s StageStore) LoadJobs(ss []*Stage) error {
	if len(ss) == 0 {
		return nil
	}

	models := interfaceSlice(len(ss), stageToInterface(ss))

	jobs := JobStore{
		Store: s.Store,
		Build: s.Build,
	}

	jj, err := jobs.All(query.WhereIn("stage_id", mapKey("id", models)...))

	if err != nil {
		return errors.Err(err)
	}

	for _, st := range ss {
		for _, j := range jj {
			if st.ID == j.StageID {
				st.Jobs = append(st.Jobs, j)
			}
		}
	}

	return nil
}

func (s StageStore) New() *Stage {
	st := &Stage{
		Model: Model{
			DB: s.DB,
		},
		Build: s.Build,
	}

	if st.Build != nil {
		st.BuildID = s.Build.ID
	}

	return st
}

func (s StageStore) Update(ss ...*Stage) error {
	models := interfaceSlice(len(ss), stageToInterface(ss))

	return errors.Err(s.Store.Update(StageTable, models...))
}
