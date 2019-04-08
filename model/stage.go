package model

import (
	"github.com/andrewpillar/thrall/errors"

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
