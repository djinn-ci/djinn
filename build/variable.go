package build

import (
	"encoding/json"

	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/variable"
)

type Variable struct {
	ID         int64
	BuildID    int64
	VariableID database.Null[int64]
	Key        string
	Value      string
	Masked     bool

	Build    *Build
	Variable *variable.Variable
}

var _ database.Model = (*Variable)(nil)

func (v *Variable) Primary() (string, any) { return "id", v.ID }

func (v *Variable) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":          &v.ID,
		"build_id":    &v.BuildID,
		"variable_id": &v.VariableID,
		"key":         &v.Key,
		"value":       &v.Value,
		"masked":      &v.Masked,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}

	v.Variable = &variable.Variable{
		ID:     v.VariableID.Elem,
		Key:    v.Key,
		Value:  v.Value,
		Masked: v.Masked,
	}
	return nil
}

func (v *Variable) Params() database.Params {
	return database.Params{
		"id":          database.ImmutableParam(v.ID),
		"build_id":    database.CreateOnlyParam(v.BuildID),
		"variable_id": database.CreateOnlyParam(v.VariableID),
		"key":         database.CreateOnlyParam(v.Key),
		"value":       database.CreateOnlyParam(v.Value),
		"masked":      database.CreateOnlyParam(v.Masked),
	}
}

func (v *Variable) Bind(m database.Model) {
	switch v2 := m.(type) {
	case *Build:
		if v.BuildID == v2.ID {
			v.Build = v2
		}
	case *variable.Variable:
		if v.VariableID.Elem == v2.ID {
			v.Variable = v2
		}
	}
}

func (*Variable) Endpoint(...string) string { return "" }

func (v *Variable) MarshalJSON() ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}

	value := variable.MaskString

	if !v.Masked {
		value = v.Value
	}

	raw := map[string]any{
		"id":           v.ID,
		"build_id":     v.BuildID,
		"key":          v.Key,
		"value":        value,
		"masked":       v.Masked,
		"build":        v.Build,
		"variable_url": nil,
	}

	if v.Variable != nil {
		raw["variable_url"] = env.DJINN_API_SERVER + v.Variable.Endpoint()
	}

	b, err := json.Marshal(raw)

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

type VariableStore struct {
	*database.Store[*Variable]
}

func NewVariableStore(pool *database.Pool) *database.Store[*Variable] {
	return database.NewStore[*Variable](pool, "build_variables", func() *Variable {
		return &Variable{}
	})
}
