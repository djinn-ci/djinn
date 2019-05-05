package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type Variable struct {
	model

	UserID       int64        `db:"user_id"`
	Key          string       `db:"key"`
	Value        string       `db:"value"`
	FromManifest bool         `db:"from_manifest"`
	DeletedAt    *pq.NullTime `db:"deleted_at"`

	User  *User
}

type VariableStore struct {
	*Store

	user *User
}

func (vs VariableStore) New() *Variable {
	v := &Variable{
		model: model{
			DB: vs.DB,
		},
		User:  vs.user,
	}

	if vs.user != nil {
		v.UserID = vs.user.ID
	}

	return v
}

func (vs VariableStore) All() ([]*Variable, error) {
	vv := make([]*Variable, 0)

	query := "SELECT * FROM variables"
	args := []interface{}{}

	if vs.user != nil {
		query += " WHERE user_id = $1"
		args = append(args, vs.user.ID)
	}

	err := vs.Select(&vv, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, v := range vv {
		v.DB = vs.DB

		if vs.user != nil {
			v.User = vs.user
		}
	}

	return vv, errors.Err(err)
}

func (vs VariableStore) In(ids ...int64) ([]*Variable, error) {
	vv := make([]*Variable, 0)

	if len(ids) == 0 {
		return vv, nil
	}

	query, args, err := sqlx.In("SELECT * FROM variables WHERE id in (?)", ids)

	if err != nil {
		return vv, errors.Err(err)
	}

	err = vs.Select(&vv, vs.Rebind(query), args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, v := range vv {
		v.DB = vs.DB
	}

	return vv, nil
}

func (v *Variable) Create() error {
	stmt, err := v.Prepare(`
		INSERT INTO variables (user_id, key, value, from_manifest)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(v.UserID, v.Key, v.Value, v.FromManifest)

	return errors.Err(row.Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt))
}