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
	StartedAt  *pq.NullTime  `db:"started_at"`
	FinishedAt *pq.NullTime  `db:"finished_at"`

	Build *Build
	Jobs  []*Job
}

type StageStore struct {
	*sqlx.DB

	Build *Build
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

func (s *Stage) JobStore() JobStore {
	return JobStore{
		DB:    s.DB,
		Build: s.Build,
		Stage: s,
	}
}

func (s *Stage) Update() error {
	stmt, err := s.Prepare(`
		UPDATE stages
		SET status = $1, started_at = $2, finished_at = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(s.Status, s.StartedAt, s.FinishedAt, s.ID)

	return errors.Err(row.Scan(&s.UpdatedAt))
}

func (stgs StageStore) All() ([]*Stage, error) {
	ss := make([]*Stage, 0)

	query := "SELECT * FROM stages"
	args := []interface{}{}

	if stgs.Build != nil {
		query += " WHERE build_id = $1"
		args = append(args, stgs.Build.ID)
	}

	err := stgs.Select(&ss, query, args...)

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

	query := "SELECT * FROM stages WHERE id = $1"
	args := []interface{}{id}

	if stgs.Build != nil {
		query += " AND build_id = $2"
		args = append(args, stgs.Build.ID)

		s.Build = stgs.Build
	}

	err := stgs.Get(s, query, args...)

	if err == sql.ErrNoRows {
		err = nil

		s.CreatedAt = nil
		s.UpdatedAt = nil
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

	query := "SELECT * FROM stages WHERE name = $1"
	args := []interface{}{name}

	if stgs.Build != nil {
		query += " AND build_id = $2"
		args = append(args, stgs.Build.ID)
	}

	err := stgs.Get(s, query, args...)

	if err == sql.ErrNoRows {
		err = nil

		s.CreatedAt = nil
		s.UpdatedAt = nil
	}

	return s, errors.Err(err)
}

func (stgs StageStore) LoadJobs(ss []*Stage) error {
	if len(ss) == 0 {
		return nil
	}

	ids := make([]int64, len(ss), len(ss))

	for i, s := range ss {
		ids[i] = s.ID
	}

	jobs := JobStore{
		DB: stgs.DB,
	}

	jj, err := jobs.InStageID(ids...)

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
