package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type Stage struct {
	model

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
	*sqlx.DB

	Build *Build
}

func (s *Stage) Create() error {
	q := Insert(
		Table("stages"),
		Columns("build_id", "name", "can_fail"),
		Values(s.BuildID, s.Name, s.CanFail),
		Returning("id", "created_at", "updated_at"),
	)

	stmt, err := s.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt))
}

func (s *Stage) JobStore() JobStore {
	return JobStore{
		DB:    s.DB,
		Build: s.Build,
		Stage: s,
	}
}

func (s *Stage) Update() error {
	q := Update(
		Table("stages"),
		Set("status", s.Status),
		Set("started_at", s.StartedAt),
		Set("finished_at", s.FinishedAt),
		SetRaw("updated_at", "NOW()"),
		WhereEq("id", s.ID),
		Returning("updated_at"),
	)

	stmt, err := s.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&s.UpdatedAt))
}

func (stgs StageStore) All(opts ...Option) ([]*Stage, error) {
	ss := make([]*Stage, 0)

	opts = append([]Option{Columns("*")}, opts...)

	q := Select(append(opts, ForBuild(stgs.Build), Table("stages"))...)

	err := stgs.Select(&ss, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, s := range ss {
		s.DB = stgs.DB

		if stgs.Build != nil {
			s.Build = stgs.Build
		}
	}

	return ss, errors.Err(err)
}

func (stgs StageStore) Find(id int64) (*Stage, error) {
	s := &Stage{
		model: model{
			DB: stgs.DB,
		},
	}

	q := Select(
		Columns("*"),
		Table("stages"),
		WhereEq("id", id),
		ForBuild(stgs.Build),
	)

	err := stgs.Get(s, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return s, errors.Err(err)
}

func (stgs StageStore) FindByName(name string) (*Stage, error) {
	s := &Stage{
		model: model{
			DB: stgs.DB,
		},
		Build: stgs.Build,
	}

	q := Select(
		Columns("*"),
		Table("stages"),
		WhereEq("name", name),
		ForBuild(stgs.Build),
	)

	err := stgs.Get(s, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return s, errors.Err(err)
}

func (stgs StageStore) LoadJobs(ss []*Stage) error {
	if len(ss) == 0 {
		return nil
	}

	ids := make([]interface{}, len(ss), len(ss))

	for i, s := range ss {
		ids[i] = s.ID
	}

	jobs := JobStore{
		DB: stgs.DB,
	}

	jj, err := jobs.All(WhereIn("stage_id", ids...))

	if err != nil {
		return errors.Err(err)
	}

	for _, s := range ss {
		for _, j := range jj {
			if s.ID == j.StageID {
				s.Jobs = append(s.Jobs, j)
			}
		}
	}

	return nil
}

func (stgs StageStore) New() *Stage {
	s := &Stage{
		model: model{
			DB: stgs.DB,
		},
		Build: stgs.Build,
	}

	if stgs.Build != nil {
		s.BuildID = stgs.Build.ID
	}

	return s
}
