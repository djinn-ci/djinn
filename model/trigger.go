package model

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"
	"github.com/andrewpillar/thrall/model/types"

	"github.com/jmoiron/sqlx"
)

type triggerData struct {
	User   string
	Email  string
	Source string
}

type Trigger struct {
	Model

	BuildID int64         `db:"build_id"`
	Type    types.Trigger `db:"type"`
	Comment string        `db:"comment"`
	Data    triggerData   `db:"data"`

	Build *Build
}

func (t *triggerData) Scan(val interface{}) error {
	b, err := types.Scan(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b) == 0 {
		return nil
	}

	buf := bytes.NewBuffer(b)
	dec := json.NewDecoder(buf)

	return errors.Err(dec.Decode(t))
}

func (t triggerData) Value() (driver.Value, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)

	if err := enc.Encode(t); err != nil {
		return driver.Value(""), errors.Err(err)
	}

	return driver.Value(buf.String()), nil
}

type TriggerStore struct {
	*sqlx.DB

	Build *Build
}

func (t *Trigger) Create() error {
	q := query.Insert(
		query.Table("triggers"),
		query.Columns("build_id", "type", "comment", "data"),
		query.Values(t.BuildID, t.Type, t.Comment, t.Data),
		query.Returning("id", "created_at", "updated_at"),
	)

	stmt, err := t.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt))
}

func (ts TriggerStore) First() (*Trigger, error) {
	t := &Trigger{
		Model: Model{
			DB: ts.DB,
		},
		Build: ts.Build,
	}

	q := query.Select(
		query.Columns("*"),
		query.Table("triggers"),
		ForBuild(ts.Build),
	)

	err := ts.Get(t, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return t, errors.Err(err)
}

func (ts TriggerStore) New() *Trigger {
	t := &Trigger{
		Model: Model{
			DB:    ts.DB,
		},
		Build: ts.Build,
	}

	if ts.Build != nil {
		t.BuildID = ts.Build.ID
	}

	return t
}
