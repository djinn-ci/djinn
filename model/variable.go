package model

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"

	"github.com/jmoiron/sqlx"
)

var _ Resource = Variable{}

type Variable struct {
	Model

	UserID      int64         `db:"user_id"`
	NamespaceID sql.NullInt64 `db:"namespace_id"`
	Key         string        `db:"key"`
	Value       string        `db:"value"`

	User      *User
	Namespace *Namespace
}

type BuildVariable struct {
	Model

	BuildID      int64         `db:"build_id"`
	VariableID   sql.NullInt64 `db:"variable_id"`
	Key          string        `db:"key"`
	Value        string        `db:"value"`

	Build    *Build
	Variable *Variable
}

type VariableStore struct {
	*sqlx.DB

	User      *User
	Namespace *Namespace
}

type BuildVariableStore struct {
	*sqlx.DB

	Build    *Build
	Variable *Variable
}

func (bvs BuildVariableStore) All(opts ...query.Option) ([]*BuildVariable, error) {
	vv := make([]*BuildVariable, 0)

	opts = append([]query.Option{query.Columns("*")}, opts...)
	opts = append(opts, ForBuild(bvs.Build), query.Table("build_variables"))

	q := query.Select(opts...)

	err := bvs.Select(&vv, q.Build(), q.Args()...)

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

	ids := make([]interface{}, 0, len(bvv))

	for _, bv := range bvv {
		if bv.VariableID.Valid {
			ids = append(ids, bv.VariableID.Int64)
		}
	}

	vv, err := variables.All(query.WhereIn("id", ids...))

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
		Model: Model{
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
	q := query.Insert(
		query.Table("variables"),
		query.Columns("user_id", "namespace_id", "key", "value"),
		query.Values(v.UserID, v.NamespaceID, v.Key, v.Value),
		query.Returning("id", "created_at", "updated_at"),
	)

	stmt, err := v.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt))
}

func (v *Variable) Destroy() error {
	q := query.Update(
		query.Table("build_variables"),
		query.SetRaw("variable_id", "NULL"),
		query.WhereEq("variable_id", v.ID),
	)

	stmt1, err := v.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt1.Close()

	if _, err := stmt1.Exec(q.Args()...); err != nil {
		return errors.Err(err)
	}

	q = query.Delete(
		query.Table("variables"),
		query.WhereEq("id", v.ID),
	)

	stmt2, err := v.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt2.Close()

	_, err = stmt2.Exec(q.Args()...)

	return errors.Err(err)
}

func (v Variable) IsZero() bool {
	return v.Model.IsZero() && v.UserID == 0 && v.Key == "" && v.Value == ""
}

func (v Variable) AccessibleBy(u *User, a Action) bool {
	if u == nil {
		return false
	}

	return v.UserID == u.ID
}

func (v Variable) UIEndpoint(uri ...string) string {
	endpoint := fmt.Sprintf("/variables/%v", v.ID)

	if len(uri) > 0 {
		endpoint = fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}

	return endpoint
}

func (vs VariableStore) All(opts ...query.Option) ([]*Variable, error) {
	vv := make([]*Variable, 0)

	opts = append([]query.Option{query.Columns("*")}, opts...)
	opts = append(opts, ForUser(vs.User), ForNamespace(vs.Namespace), query.Table("variables"))

	q := query.Select(opts...)

	err := vs.Select(&vv, q.Build(), q.Args()...)

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

func (vs VariableStore) findBy(col string, val interface{}) (*Variable, error) {
	v := &Variable{
		Model: Model{
			DB: vs.DB,
		},
		User: vs.User,
	}

	q := query.Select(
		query.Columns("*"),
		query.Table("variables"),
		query.WhereEq(col, val),
		ForUser(vs.User),
		ForNamespace(vs.Namespace),
	)

	err := vs.Get(v, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return v, errors.Err(err)
}

func (vs VariableStore) Find(id int64) (*Variable, error) {
	v, err := vs.findBy("id", id)

	return v, errors.Err(err)
}

func (vs VariableStore) FindByKey(key string) (*Variable, error) {
	v, err := vs.findBy("key", key)

	return v, errors.Err(err)
}

func (vs VariableStore) Index(opts ...query.Option) ([]*Variable, error) {
	vv, err := vs.All(opts...)

	if err != nil {
		return vv, errors.Err(err)
	}

	namespaces := NamespaceStore{
		DB: vs.DB,
	}

	ids := make([]interface{}, len(vv), len(vv))

	for i, v := range vv {
		if v.NamespaceID.Valid {
			ids[i] = v.NamespaceID.Int64
		}
	}

	nn := make([]*Namespace, 0, len(ids))
	userIds := make([]interface{}, 0, len(ids))

	err = namespaces.Load(ids, func(i int, n *Namespace) {
		v := vv[i]

		if v.NamespaceID.Int64 == n.ID {
			nn = append(nn, n)
			userIds = append(userIds, n.UserID)

			v.Namespace = n
		}
	})

	if err != nil {
		return vv, errors.Err(err)
	}

	users := UserStore{
		DB: vs.DB,
	}

	err = users.Load(userIds, func(i int, u *User) {
		n := nn[i]

		if n.UserID == u.ID {
			n.User = u
		}
	})

	return vv, errors.Err(err)
}

func (vs VariableStore) New() *Variable {
	v := &Variable{
		Model: Model{
			DB: vs.DB,
		},
		User:  vs.User,
	}

	if vs.User != nil {
		v.UserID = vs.User.ID
	}

	return v
}
