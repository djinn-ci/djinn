package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"
)

type Driver struct {
	model

	BuildID int64      `db:"build_id"`
	Type    DriverType `db:"type"`
	Config  string     `db:"config"`

	Build *Build
}

type DriverStore struct {
	*sqlx.DB

	Build *Build
}

func (d *Driver) Create() error {
	q := Insert(
		Table("drivers"),
		Columns("build_id", "type", "config"),
		Values(d.BuildID, d.Type, d.Config),
		Returning("id", "created_at", "updated_at"),
	)

	stmt, err := d.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt))
}

func (ds DriverStore) First() (*Driver, error) {
	d := &Driver{
		model: model{
			DB: ds.DB,
		},
		Build: ds.Build,
	}

	q := Select(
		Columns("*"),
		Table("drivers"),
		ForBuild(ds.Build),
	)

	err := ds.Get(d, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return d, errors.Err(err)
}

func (ds DriverStore) New() *Driver {
	d := &Driver{
		model: model{
			DB: ds.DB,
		},
		Build: ds.Build,
	}

	if ds.Build != nil {
		d.BuildID = ds.Build.ID
	}

	return d
}
