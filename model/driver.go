package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"
	"github.com/andrewpillar/thrall/model/types"
)

type Driver struct {
	Model

	BuildID int64        `db:"build_id"`
	Type    types.Driver `db:"type"`
	Config  string       `db:"config"`

	Build *Build
}

type DriverStore struct {
	Store

	Build *Build
}

func (d Driver) Values() map[string]interface{} {
	return map[string]interface{}{
		"build_id": d.BuildID,
		"type":     d.Type,
		"config":   d.Config,
	}
}

func (s DriverStore) interfaceSlice(dd ...*Driver) []Interface {
	ii := make([]Interface, len(dd), len(dd))

	for i, d := range dd {
		ii[i] = d
	}

	return ii
}

func (s DriverStore) Create(dd ...*Driver) error {
	return errors.Err(s.Store.Create(DriverTable, s.interfaceSlice(dd...)...))
}

func (s DriverStore) First() (*Driver, error) {
	d := &Driver{
		Model: Model{
			DB: s.DB,
		},
		Build: s.Build,
	}

	q := query.Select(
		query.Columns("*"),
		query.Table(DriverTable),
		ForBuild(s.Build),
	)

	err := s.Store.Get(d, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return d, errors.Err(err)
}

func (s DriverStore) New() *Driver {
	d := &Driver{
		Model: Model{
			DB: s.DB,
		},
		Build: s.Build,
	}

	if s.Build != nil {
		d.BuildID = s.Build.ID
	}

	return d
}
