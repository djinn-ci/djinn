package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type Stage struct {
	Model

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

func StagesByBuildID(id int64) ([]*Stage, error) {
	stages := make([]*Stage, 0)

	err := DB.Select(&stages, "SELECT * FROM stages WHERE build_id = $1", id)

	if err != nil {
		if err == sql.ErrNoRows {
			return stages, nil
		}

		return stages, errors.Err(err)
	}

	return stages, nil
}

func LoadStageJobs(stages []*Stage) error {
	stageIds := make([]int64, len(stages), len(stages))

	for i, s := range stages {
		stageIds[i] = s.ID
	}

	query, args, err := sqlx.In("SELECT * FROM jobs WHERE stage_id IN (?)", stageIds)

	if err != nil {
		return errors.Err(err)
	}

	jobs := make([]*Job, 0)

	err = DB.Select(&jobs, DB.Rebind(query), args...)

	if err != nil {
		return errors.Err(err)
	}

	for _, s := range stages {
		if s.Jobs == nil {
			s.Jobs = make([]*Job, 0, len(jobs))
		}

		for _, j := range jobs {

			if j.StageID == s.ID {
				s.Jobs = append(s.Jobs, j)
			}
		}
	}

	return nil
}

func (s *Stage) Create() error {
	stmt, err := DB.Prepare(`
		INSERT INTO stages (build_id, name, can_fail)
		VALUES ($1, $2, $3)
		RETURNING id
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	err = stmt.QueryRow(s.BuildID, s.Name, s.CanFail).Scan(&s.ID)

	return errors.Err(err)
}
