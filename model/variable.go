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

type BuildVariable struct {
	model

	BuildID    int64 `db:"build_id"`
	VariableID int64 `db:"variable_id"`

	Build    *Build
	Variable *Variable
}

type VariableStore struct {
	*sqlx.DB

	user *User
}

type BuildVariableStore struct {
	*sqlx.DB

	build    *Build
	variable *Variable
}

func (bvs BuildVariableStore) All() ([]*BuildVariable, error) {
	vv := make([]*BuildVariable, 0)

	query := "SELECT * FROM build_variables"
	args := []interface{}{}

	if bvs.build != nil {
		query += " WHERE build_id = $1"
		args = append(args, bvs.build.ID)
	}

	err := bvs.Select(&vv, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, v := range vv {
		v.DB = bvs.DB
		v.Build = bvs.build
	}

	return vv, errors.Err(err)
}

func (bvs BuildVariableStore) LoadVariables(bvv []*BuildVariable) error {
	if len(bvv) == 0 {
		return nil
	}

	variables := VariableStore{
		DB: bvs.DB,
	}

	ids := make([]int64, len(bvv), len(bvv))

	for i, bv := range bvv {
		ids[i] = bv.VariableID
	}

	vv, err := variables.In(ids...)

	if err != nil {
		return errors.Err(err)
	}

	for _, v := range vv {
		for _, bv := range bvv {
			if v.ID == bv.VariableID {
				bv.Variable = v
			}
		}
	}

	return nil
}

func (bvs BuildVariableStore) New() *BuildVariable {
	bv := &BuildVariable{
		model: model{
			DB: bvs.DB,
		},
		Build: bvs.build,
	}

	if bvs.build != nil {
		bv.BuildID = bvs.build.ID
	}

	return bv
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
