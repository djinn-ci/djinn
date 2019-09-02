package model

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"
)

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
	Store

	User      *User
	Namespace *Namespace
}

type BuildVariableStore struct {
	Store

	Build    *Build
	Variable *Variable
}

func buildVariableToInterface(bvv []*BuildVariable) func(i int) Interface {
	return func(i int) Interface {
		return bvv[i]
	}
}

func variableToInterface(vv []*Variable) func(i int) Interface {
	return func(i int) Interface {
		return vv[i]
	}
}

func (b BuildVariable) Values() map[string]interface{} {
	return map[string]interface{}{
		"build_id":    b.BuildID,
		"variable_id": b.VariableID,
		"key":         b.Key,
		"value":       b.Value,
	}
}

func (s BuildVariableStore) All(opts ...query.Option) ([]*BuildVariable, error) {
	vv := make([]*BuildVariable, 0)

	opts = append(opts, ForBuild(s.Build))

	err := s.Store.All(&vv, BuildVariableTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, v := range vv {
		v.DB = s.DB
		v.Build = s.Build
	}

	return vv, errors.Err(err)
}

func (s BuildVariableStore) Create(bvv ...*BuildVariable) error {
	models := interfaceSlice(len(bvv), buildVariableToInterface(bvv))

	return errors.Err(s.Store.Create(BuildVariableTable, models...))
}

func (s BuildVariableStore) Copy(vv []*Variable) error {
	if len(vv) == 0 {
		return nil
	}

	bvv := make([]*BuildVariable, 0, len(vv))

	for _, v := range vv {
		bv := s.New()
		bv.VariableID = sql.NullInt64{
			Int64: v.ID,
			Valid: true,
		}
		bv.Key = v.Key
		bv.Value = v.Value

		bvv = append(bvv, bv)
	}

	return errors.Err(s.Create(bvv...))
}

func (s BuildVariableStore) LoadVariables(bvv []*BuildVariable) error {
	if len(bvv) == 0 {
		return nil
	}

	variables := VariableStore{
		Store: s.Store,
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

func (s BuildVariableStore) New() *BuildVariable {
	bv := &BuildVariable{
		Model: Model{
			DB: s.DB,
		},
		Build:    s.Build,
		Variable: s.Variable,
	}

	if s.Build != nil {
		bv.BuildID = s.Build.ID
	}

	if s.Variable != nil {
		bv.VariableID = sql.NullInt64{
			Int64: s.Variable.ID,
			Valid: true,
		}
	}

	return bv
}

func (v Variable) IsZero() bool {
	return v.Model.IsZero() && v.UserID == 0 && v.Key == "" && v.Value == ""
}

func (v Variable) UIEndpoint(uri ...string) string {
	endpoint := fmt.Sprintf("/variables/%v", v.ID)

	if len(uri) > 0 {
		endpoint = fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}

	return endpoint
}

func (v Variable) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":      v.UserID,
		"namespace_id": v.NamespaceID,
		"key":          v.Key,
		"value":        v.Value,
	}
}

func (s VariableStore) All(opts ...query.Option) ([]*Variable, error) {
	vv := make([]*Variable, 0)

	opts = append(opts, ForCollaborator(s.User), ForNamespace(s.Namespace))

	err := s.Store.All(&vv, VariableTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, v := range vv {
		v.DB = s.DB

		if s.User != nil {
			v.User = s.User
		}
	}

	return vv, errors.Err(err)
}

func (s VariableStore) Create(vv ...*Variable) error {
	models := interfaceSlice(len(vv), variableToInterface(vv))

	return errors.Err(s.Store.Create(VariableTable, models...))
}

func (s VariableStore) Delete(vv ...*Variable) error {
	models := interfaceSlice(len(vv), variableToInterface(vv))

	return errors.Err(s.Store.Delete(VariableTable, models...))
}

func (s VariableStore) findBy(col string, val interface{}) (*Variable, error) {
	v := &Variable{
		Model: Model{
			DB: s.DB,
		},
		User: s.User,
	}

	err := s.FindBy(v, VariableTable, col, val)

	if err == sql.ErrNoRows {
		err = nil
	}

	return v, errors.Err(err)
}

func (s VariableStore) Find(id int64) (*Variable, error) {
	v, err := s.findBy("id", id)

	return v, errors.Err(err)
}

func (s VariableStore) FindByKey(key string) (*Variable, error) {
	v, err := s.findBy("key", key)

	return v, errors.Err(err)
}

func (s VariableStore) Index(opts ...query.Option) ([]*Variable, error) {
	vv, err := s.All(opts...)

	if err != nil {
		return vv, errors.Err(err)
	}

	if err := s.LoadNamespaces(vv); err != nil {
		return vv, errors.Err(err)
	}

	nn := make([]*Namespace, 0, len(vv))

	for _, v := range vv {
		if v.Namespace != nil {
			nn = append(nn, v.Namespace)
		}
	}

	namespaces := NamespaceStore{
		Store: s.Store,
	}

	err = namespaces.LoadUsers(nn)

	return vv, errors.Err(err)
}

func (s VariableStore) interfaceSlice(vv ...*Variable) []Interface {
	ii := make([]Interface, len(vv), len(vv))

	for i, v := range vv {
		ii[i] = v
	}

	return ii
}

func (s VariableStore) loadNamespace(vv []*Variable) func(i int, n *Namespace) {
	return func(i int, n *Namespace) {
		v := vv[i]

		if v.NamespaceID.Int64 == n.ID {
			v.Namespace = n
		}
	}
}

func (s VariableStore) LoadNamespaces(vv []*Variable) error {
	if len(vv) == 0 {
		return nil
	}

	models := interfaceSlice(len(vv), variableToInterface(vv))

	namespaces := NamespaceStore{
		Store: s.Store,
	}

	err := namespaces.Load(mapKey("namespace_id", models), s.loadNamespace(vv))

	return errors.Err(err)
}

func (s VariableStore) New() *Variable {
	v := &Variable{
		Model: Model{
			DB: s.DB,
		},
		User:  s.User,
	}

	if s.User != nil {
		v.UserID = s.User.ID
	}

	return v
}
