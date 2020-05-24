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
	Variable *variable.Variable
}

var (
	_ model.Model  = (*Variable)(nil)
	_ model.Binder = (*VariableStore)(nil)
	_ model.Loader = (*VariableStore)(nil)

	variableTable = "build_variables"
)

// NewVariableStore returns a new VariableStore for querying the build_variables
// table. Each model passed to this function will be bound to the returned
// VariableStore.
func NewVariableStore(db *sqlx.DB, mm ...model.Model) *VariableStore {
	s := &VariableStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// VariableModel is called along with model.Slice to convert the given slice of
// Variable models to a slice of model.Model interfaces.
func VariableModel(vv []*Variable) func(int) model.Model {
	return func(i int) model.Model {
		return vv[i]
	}
}

// Bind the given models to the current Variable. This will only bind the model
// if they are one of the following,
//
// - *Build
// - *variable.Variable
func (v *Variable) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			v.Build = m.(*Build)
		case *variable.Variable:
			v.Variable = m.(*variable.Variable)
		}
	}
}

func (v *Variable) SetPrimary(id int64) {
	v.ID = id
}

func (v Variable) Primary() (string, int64) {
	return "id", v.ID
}

func (v *Variable) IsZero() bool {
	return v == nil || v.ID == 0 &&
		v.BuildID == 0 &&
		!v.VariableID.Valid &&
		v.Key == "" &&
		v.Value == ""
}

func (v *Variable) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":       v.ID,
		"build_id": v.BuildID,
		"key":      v.Key,
		"value":    v.Value,
	}

	if !v.Build.IsZero() {
		json["build"] = v.Build.JSON(addr)
	}

	if !v.Variable.IsZero() {
		json["variable_url"] = addr + v.Variable.Endpoint()
	}
	return json
}

// Endpoint is a stub to fulfill the model.Model interface. It returns an empty
// string.
func (v *Variable) Endpoint(_ ...string) string { return "" }

func (v Variable) Values() map[string]interface{} {
	return map[string]interface{}{
		"build_id":    v.BuildID,
		"variable_id": v.VariableID,
		"key":         v.Key,
		"value":       v.Value,
	}
}

// New returns a new Variable binding any non-nil models to it from the current
// VariableStore.
func (s VariableStore) New() *Variable {
	v := &Variable{
		Build: s.Build,
	}

	if s.Build != nil {
		v.BuildID = s.Build.ID
	}
	return v
}

// Bind the given models to the current VariableStore. This will only bind the
// model if they are one of the following,
//
// - *Build
// - *variable.Variable
func (s *VariableStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			s.Build = m.(*Build)
		case *variable.Variable:
			s.Variable = m.(*variable.Variable)
		}
	}
}

// Create inserts the given Variable models into the build_variables table.
func (s VariableStore) Create(vv ...*Variable) error {
	models := model.Slice(len(vv), VariableModel(vv))
	return errors.Err(s.Store.Create(variableTable, models...))
}

// All returns a slice of Variable models, applying each query.Option that is
// given. The model.Where option is used on the Build and Variable bound models
// to limit the query to those relations.
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

// Load loads in a slice of Variable models where the given key is in the list
// of given vals. Each model is loaded individually via a call to the given
// load callback. This method calls StageStore.All under the hood, so any bound
// models will impact the models being loaded.
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
