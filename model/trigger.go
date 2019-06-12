package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"
)

type Trigger struct {
	model

	BuildID int64       `db:"build_id"`
	Type    TriggerType `db:"type"`
	Comment string      `db:"comment"`
	Data    string      `db:"data"`

	Build *Build
}

type TriggerStore struct {
	*sqlx.DB

	Build *Build
}

func (t *Trigger) Create() error {
	stmt, err := t.Prepare(`
		INSERT INTO triggers (build_id, type, comment, data)
		VALUES ($1, $2, $3, '{}')
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(t.BuildID, t.Type, t.Comment)

	return errors.Err(row.Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt))
}

func (ts TriggerStore) First() (*Trigger, error) {
	t := &Trigger{
		model: model{
			DB: ts.DB,
		},
		Build: ts.Build,
	}

	query := "SELECT * FROM triggers"
	args := []interface{}{}

	if ts.Build != nil {
		query += " WHERE build_id = $1"
		args = append(args, ts.Build.ID)
	}

	err := ts.Get(t, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return t, errors.Err(err)
}

func (ts TriggerStore) New() *Trigger {
	t := &Trigger{
		model: model{
			DB:    ts.DB,
		},
		Build: ts.Build,
	}

	if ts.Build != nil {
		t.BuildID = ts.Build.ID
	}

	return t
}
