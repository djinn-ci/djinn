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
	stmt, err := d.Prepare(`
		INSERT INTO drivers (build_id, type, config)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(d.BuildID, d.Type, d.Config)

	return errors.Err(row.Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt))
}

func (ds DriverStore) First() (*Driver, error) {
	d := &Driver{
		model: model{
			DB: ds.DB,
		},
		Build: ds.Build,
	}

	query := "SELECT * FROM drivers"
	args := []interface{}{}

	if ds.Build != nil {
		query += " WHERE build_id = $1"
		args = append(args, ds.Build.ID)
	}

	err := ds.Get(d, query, args...)

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
