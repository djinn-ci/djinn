package build

import (
	"database/sql"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Variable is the type that represents a variable that has been set on a build.
type Variable struct {
	ID         int64         `db:"id"`
	BuildID    int64         `db:"build_id"`
	VariableID sql.NullInt64 `db:"variable_id"`
	Key        string        `db:"key"`
	Value      string        `db:"value"`

	Build    *Build             `db:"-"`
	Variable *variable.Variable `db:"-"`
}

// VariableStore is the type for creating and modifying Variable models in the
// database.
type VariableStore struct {
	database.Store

	Build    *Build
	Variable *variable.Variable
}

var (
	_ database.Model  = (*Variable)(nil)
	_ database.Binder = (*VariableStore)(nil)
	_ database.Loader = (*VariableStore)(nil)

	variableTable = "build_variables"
)

// NewVariableStore returns a new VariableStore for querying the build_variables
// table. Each database passed to this function will be bound to the returned
// VariableStore.
func NewVariableStore(db *sqlx.DB, mm ...database.Model) *VariableStore {
	s := &VariableStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// VariableModel is called along with database.ModelSlice to convert the given slice of
// Variable models to a slice of database.Model interfaces.
func VariableModel(vv []*Variable) func(int) database.Model {
	return func(i int) database.Model {
		return vv[i]
	}
}

// Bind implements the database.Binder interface. This will only bind the models
// if they are pointers to either a Build or variable.Variable model.
func (v *Variable) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v1 := m.(type) {
		case *Build:
			v.Build = v1
		case *variable.Variable:
			v.Variable = v1
		}
	}
}

// SetPrimary implements the database.Model interface.
func (v *Variable) SetPrimary(id int64) { v.ID = id }

// Primary implements the database.Model interface.
func (v Variable) Primary() (string, int64) { return "id", v.ID }

// IsZero implements the database.Model interface.
func (v *Variable) IsZero() bool {
	return v == nil || v.ID == 0 &&
		v.BuildID == 0 &&
		!v.VariableID.Valid &&
		v.Key == "" &&
		v.Value == ""
}

// JSON implements the database.Model interface. This will return a map with
// the current Variable's values. If the Build bound model exists on the
// current Variable then the JSON representation will be in the returned map
// under the build key.
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

// Endpoint implements the database.Model interface. This will return an empty
// string.
func (v *Variable) Endpoint(_ ...string) string { return "" }

// Values implements the database.Model interface. This will return a map with
// the following values, build_id, variable_id, key, and value.
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
		Build:    s.Build,
		Variable: s.Variable,
	}

	if s.Build != nil {
		v.BuildID = s.Build.ID
	}

	if s.Variable != nil {
		v.VariableID = sql.NullInt64{
			Int64: v.Variable.ID,
			Valid: true,
		}
	}
	return v
}

// Bind implements the database.Binder interface. This will only bind the models
// if they are pointers to either Build or variable.Variable.
func (s *VariableStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *Build:
			s.Build = v
		case *variable.Variable:
			s.Variable = v
		}
	}
}

// Copy copies each given variable.Variable into a build Variable, and returns
// the slice of newly created Variable models.
func (s *VariableStore) Copy(vv ...*variable.Variable) ([]*Variable, error) {
	bvv := make([]*Variable, 0, len(vv))

	for _, v := range vv {
		s.Bind(v)

		bv := s.New()
		bv.Key = v.Key
		bv.Value = v.Value

		bvv = append(bvv, bv)
	}

	s.Variable = nil

	err := s.Store.Create(variableTable, database.ModelSlice(len(bvv), VariableModel(bvv))...)
	return bvv, errors.Err(err)
}

// Create creates a new Variable with the given key and val.
func (s *VariableStore) Create(key, val string) (*Variable, error) {
	v := s.New()
	v.Key = key
	v.Value = val

	err := s.Store.Create(variableTable, v)
	return v, errors.Err(err)
}

// All returns a slice of Variable models, applying each query.Option that is
// given.
func (s VariableStore) All(opts ...query.Option) ([]*Variable, error) {
	vv := make([]*Variable, 0)

	opts = append([]query.Option{
		database.Where(s.Build, "build_id"),
		database.Where(s.Variable, "variable_id"),
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
// of given vals. Each database is loaded individually via a call to the given
// load callback. This method calls StageStore.All under the hood, so any bound
// models will impact the models being loaded.
func (s VariableStore) Load(key string, vals []interface{}, load database.LoaderFunc) error {
	vv, err := s.All(query.Where(key, "IN", database.List(vals...)))

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
