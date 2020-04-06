package build

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/variable"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type Variable struct {
	ID         int64         `db:"id"`
	BuildID    int64         `db:"build_id"`
	VariableID sql.NullInt64 `db:"variable_id"`
	Key        string        `db:"key"`
	Value      string        `db:"value"`

	Build    *Build             `db:"-"`
	Variable *variable.Variable `db:"-"`
}

type VariableStore struct {
	model.Store

	Build    *Build
	Variable model.Model
}

var (
	_ model.Model  = (*Variable)(nil)
	_ model.Binder = (*VariableStore)(nil)
	_ model.Loader = (*VariableStore)(nil)

	variableTable = "build_variables"
)

func NewVariableStore(db *sqlx.DB, mm ...model.Model) VariableStore {
	s := VariableStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func VariableModel(vv []*Variable) func(int) model.Model {
	return func(i int) model.Model {
		return vv[i]
	}
}

func (v *Variable) Bind(mm ...model.Model) {
	if v == nil {
		return
	}

	for _, m := range mm {
		switch m.(type) {
		case *Build:
			v.Build = m.(*Build)
		case *variable.Variable:
			v.Variable = m.(*variable.Variable)
		}
	}
}

func (*Variable) Kind() string { return "build_variable" }

func (v *Variable) SetPrimary(id int64) {
	if v == nil {
		return
	}
	v.ID = id
}

func (v *Variable) Primary() (string, int64) {
	if v == nil {
		return "id", 0
	}
	return "id", v.ID
}

func (v *Variable) IsZero() bool {
	return v == nil || v.ID == 0 &&
		v.BuildID == 0 &&
		!v.VariableID.Valid &&
		v.Key == "" &&
		v.Value == ""
}

func (*Variable) Endpoint(_ ...string) string { return "" }

func (v *Variable) Values() map[string]interface{} {
	if v == nil {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"build_id":    v.BuildID,
		"variable_id": v.VariableID,
		"key":         v.Key,
		"value":       v.Value,
	}
}

func (s VariableStore) New() *Variable {
	v := &Variable{
		Build: s.Build,
	}

	if s.Build != nil {
		v.BuildID = s.Build.ID
	}
	return v
}

func (s *VariableStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.Kind() {
		case "build":
			s.Build = m.(*Build)
		case "variable":
			s.Variable = m
		}
	}
}

func (s VariableStore) Create(vv ...*Variable) error {
	models := model.Slice(len(vv), VariableModel(vv))
	return errors.Err(s.Store.Create(variableTable, models...))
}

func (s VariableStore) All(opts ...query.Option) ([]*Variable, error) {
	vv := make([]*Variable, 0)

	opts = append([]query.Option{
		model.Where(s.Build, "build_id"),
		model.Where(s.Variable, "variable_id"),
	}, opts...)

	err := s.Store.All(&vv, variableTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, v := range vv {
		v.Build = s.Build
	}
	return vv, errors.Err(err)
}

func (s VariableStore) Load(key string, vals []interface{}, load model.LoaderFunc) error {
	vv, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, v := range vv {
			load(i, v)
		}
	}
	return nil
}
