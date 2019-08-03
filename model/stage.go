package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"
	"github.com/andrewpillar/thrall/runner"

	"github.com/jmoiron/sqlx"

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
	*sqlx.DB

	Build *Build
}

func (s *Stage) Create() error {
	q := query.Insert(
		query.Table("stages"),
		query.Columns("build_id", "name", "can_fail"),
		query.Values(s.BuildID, s.Name, s.CanFail),
		query.Returning("id", "created_at", "updated_at"),
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
	q := query.Update(
		query.Table("stages"),
		query.Set("status", s.Status),
		query.Set("started_at", s.StartedAt),
		query.Set("finished_at", s.FinishedAt),
		query.SetRaw("updated_at", "NOW()"),
		query.WhereEq("id", s.ID),
		query.Returning("updated_at"),
	)

	stmt, err := s.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&s.UpdatedAt))
}

func (stgs StageStore) All(opts ...query.Option) ([]*Stage, error) {
	ss := make([]*Stage, 0)

	opts = append([]query.Option{query.Columns("*")}, opts...)
	opts = append(opts, ForBuild(stgs.Build), query.Table("stages"))

	q := query.Select(opts...)

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

func (stgs StageStore) findBy(col string, val interface{}) (*Stage, error) {
	s := &Stage{
		Model: Model{
			DB: stgs.DB,
		},
	}

	q := query.Select(
		query.Columns("*"),
		query.Table("stages"),
		query.WhereEq(col, val),
		ForBuild(stgs.Build),
	)

	err := stgs.Get(s, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return s, errors.Err(err)
}

func (stgs StageStore) Find(id int64) (*Stage, error) {
	s, err := stgs.findBy("id", id)

	return s, errors.Err(err)
}

func (stgs StageStore) FindByName(name string) (*Stage, error) {
	s, err := stgs.findBy("name", name)

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

	jj, err := jobs.All(query.WhereIn("stage_id", ids...))

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
		Model: Model{
			DB: stgs.DB,
		},
		Build: stgs.Build,
	}

	if stgs.Build != nil {
		s.BuildID = stgs.Build.ID
	}

	return s
}
