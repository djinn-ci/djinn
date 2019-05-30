package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"
)

type Variable struct {
	model

	UserID       int64        `db:"user_id"`
	Key          string       `db:"key"`
	Value        string       `db:"value"`

	User  *User
}

type BuildVariable struct {
	model

	BuildID      int64         `db:"build_id"`
	VariableID   sql.NullInt64 `db:"variable_id"`
	Key          string        `db:"key"`
	Value        string        `db:"value"`

	Build    *Build
	Variable *Variable
}

type VariableStore struct {
	*sqlx.DB

	User *User
}

type BuildVariableStore struct {
	*sqlx.DB

	Build    *Build
	Variable *Variable
}

func (bvs BuildVariableStore) All() ([]*BuildVariable, error) {
	vv := make([]*BuildVariable, 0)

	query := "SELECT * FROM build_variables"
	args := []interface{}{}

	if bvs.Build != nil {
		query += " WHERE build_id = $1"
		args = append(args, bvs.Build.ID)
	}

	err := bvs.Select(&vv, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, v := range vv {
		v.DB = bvs.DB
		v.Build = bvs.Build
	}

	return vv, errors.Err(err)
}

func (bvs BuildVariableStore) Copy(vv []*Variable) error {
	if len(vv) == 0 {
		return nil
	}

	for _, v := range vv {
		bv := bvs.New()
		bv.VariableID = sql.NullInt64{
			Int64: v.ID,
			Valid: true,
		}
		bv.Key = v.Key
		bv.Value = v.Value

		if err := bv.Create(); err != nil {
			return errors.Err(err)
		}
	}

	return nil
}

func (bvs BuildVariableStore) LoadVariables(bvv []*BuildVariable) error {
	if len(bvv) == 0 {
		return nil
	}

	variables := VariableStore{
		DB: bvs.DB,
	}

	ids := make([]int64, 0, len(bvv))

	for _, bv := range bvv {
		if bv.VariableID.Valid {
			ids = append(ids, bv.VariableID.Int64)
		}
	}

	vv, err := variables.In(ids...)

	if err != nil {
		return errors.Err(err)
	}

	for _, v := range vv {
		for _, bv := range bvv {
			if v.ID == bv.VariableID.Int64 && bv.VariableID.Valid {
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
		Build:    bvs.Build,
		Variable: bvs.Variable,
	}

	if bvs.Build != nil {
		bv.BuildID = bvs.Build.ID
	}

	if bvs.Variable != nil {
		bv.VariableID = sql.NullInt64{
			Int64: bvs.Variable.ID,
			Valid: true,
		}
	}

	return bv
}

func (v *Variable) Create() error {
	stmt, err := v.Prepare(`
		INSERT INTO variables (user_id, key, value)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(v.UserID, v.Key, v.Value)

	return errors.Err(row.Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt))
}

func (vs VariableStore) All() ([]*Variable, error) {
	vv := make([]*Variable, 0)

	query := "SELECT * FROM variables"
	args := []interface{}{}

	if vs.User != nil {
		query += " WHERE user_id = $1"
		args = append(args, vs.User.ID)
	}

	err := vs.Select(&vv, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, v := range vv {
		v.DB = vs.DB

		if vs.User != nil {
			v.User = vs.User
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
		User:  vs.User,
	}

	if vs.User != nil {
		v.UserID = vs.User.ID
	}

	return v
}
