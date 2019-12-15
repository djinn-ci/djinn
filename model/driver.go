package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/types"

	"github.com/andrewpillar/query"
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

func driverToInterface(dd ...*Driver) func(i int) Interface {
	return func(i int) Interface {
		return dd[i]
	}
}

func (d Driver) Values() map[string]interface{} {
	return map[string]interface{}{
		"build_id": d.BuildID,
		"type":     d.Type,
		"config":   d.Config,
	}
}

func (s DriverStore) Create(dd ...*Driver) error {
	models := interfaceSlice(len(dd), driverToInterface(dd...))

	return errors.Err(s.Store.Create(DriverTable, models...))
}

func (s DriverStore) Get(opts ...query.Option) (*Driver, error) {
	d := &Driver{
		Model: Model{
			DB: s.DB,
		},
		Build: s.Build,
	}

	baseOpts := []query.Option{
		query.Columns("*"),
		query.From(DriverTable),
		ForBuild(s.Build),
	}

	q := query.Select(append(baseOpts, opts...)...)

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
