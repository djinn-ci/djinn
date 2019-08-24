package model

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"
	"github.com/andrewpillar/thrall/model/types"
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

type TriggerStore struct {
	Store

	Build *Build
}

func triggerToInterface(tt ...*Trigger) func(i int) Interface {
	return func(i int) Interface {
		return tt[i]
	}
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

func (t Trigger) Values() map[string]interface{} {
	return map[string]interface{}{
		"build_id": t.BuildID,
		"type":     t.Type,
		"comment":  t.Comment,
		"data":     t.Data,
	}
}

func (s TriggerStore) Create(tt ...*Trigger) error {
	models := interfaceSlice(len(tt), triggerToInterface(tt...))

	return errors.Err(s.Store.Create(TriggerTable, models...))
}

func (s TriggerStore) First() (*Trigger, error) {
	t := &Trigger{
		Model: Model{
			DB: s.DB,
		},
		Build: s.Build,
	}

	q := query.Select(
		query.Columns("*"),
		query.Table(TriggerTable),
		ForBuild(s.Build),
	)

	err := s.Store.Get(t, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return t, errors.Err(err)
}

func (s TriggerStore) New() *Trigger {
	t := &Trigger{
		Model: Model{
			DB: s.DB,
		},
		Build: s.Build,
	}

	if s.Build != nil {
		t.BuildID = s.Build.ID
	}

	return t
}
