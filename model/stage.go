package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type Stage struct {
	model

	BuildID    int64        `db:"build_id"`
	Name       string       `db:"name"`
	CanFail    bool         `db:"can_fail"`
	DidFail    bool         `db:"did_fail"`
	Status     Status       `db:"status"`
	StartedAt  *pq.NullTime `db:"started_at"`
	FinishedAt *pq.NullTime `db:"finished_at"`

	Build *Build
	Jobs  []*Job
}

type StageStore struct {
	*Store

	build *Build
}

func (stgs StageStore) New() *Stage {
	s := &Stage{
		model: model{
			DB: stgs.DB,
		},
		Build: stgs.build,
	}

	if stgs.build != nil {
		s.BuildID = stgs.build.ID
	}

	return s
}

func (stgs StageStore) All() ([]*Stage, error) {
	ss := make([]*Stage, 0)

	query := "SELECT * FROM stages"
	args := []interface{}{}

	if stgs.build != nil {
		query += " WHERE build_id = $1"
		args = append(args, stgs.build.ID)
	}

	err := stgs.Select(&ss, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, s := range ss {
		s.DB = stgs.DB

		if stgs.build != nil {
			s.Build = stgs.build
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

	query := "SELECT * FROM stages WHERE id = $1"
	args := []interface{}{id}

	if stgs.build != nil {
		query += " AND build_id = $2"
		args = append(args, stgs.build.ID)

		s.Build = stgs.build
	}

	err := stgs.Get(s, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return s, errors.Err(err)
}

func (stgs StageStore) LoadJobs(ss []*Stage) error {
	ids := make([]int64, len(ss), len(ss))

	for i, s := range ss {
		ids[i] = s.ID
	}

	query, args, err := sqlx.In("SELECT * FROM jobs WHERE stage_id IN (?)", ids)

	if err != nil {
		return errors.Err(err)
	}

	jj := make([]*Job, 0)

	if err := stgs.Select(&jj, stgs.Rebind(query), args...); err != nil && err != sql.ErrNoRows {
		return errors.Err(err)
	}

	for _, s := range ss {
		if s.Jobs == nil {
			s.Jobs = make([]*Job, 0, len(jj))
		}

		for _, j := range jj {
			if j.StageID == s.ID {
				s.Jobs = append(s.Jobs, j)
			}
		}
	}

	return nil
}

func (s *Stage) JobStore() JobStore {
	return JobStore{
		Store: &Store{
			DB: s.DB,
		},
		build: s.Build,
		stage: s,
	}
}

func (s *Stage) Create() error {
	stmt, err := s.Prepare(`
		INSERT INTO stages (build_id, name, can_fail)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(s.BuildID, s.Name, s.CanFail)

	return errors.Err(row.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt))
}

func (s *Stage) Update() error {
	stmt, err := s.Prepare(`
		UPDATE stages
		SET did_fail = $1, status = $2, started_at = $3, finished_at = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(s.DidFail, s.Status, s.StartedAt, s.FinishedAt, s.ID)

	return errors.Err(row.Scan(&s.UpdatedAt))
}
