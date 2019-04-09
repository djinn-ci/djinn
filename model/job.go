package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"

	"github.com/lib/pq"
)

type Job struct {
	Model

	StageID    int64          `db:"stage_id"`
	Name       string         `db:"name"`
	Status     Status         `db:"status"`
	Output     sql.NullString `db:"output"`
	StartedAt  *pq.NullTime   `db:"started_at"`
	FinishedAt *pq.NullTime   `db:"finished_at"`

	Stage *Stage
}

func (j *Job) Create() error {
	stmt, err := DB.Prepare(`
		INSERT INTO jobs (stage_id, name)
		VALUES ($1, $2)
		RETURNING id
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	err = stmt.QueryRow(j.StageID, j.Name).Scan(&j.ID)

	return errors.Err(err)
}
