package build

import (
	"database/sql"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/query"
)

type Variable struct {
	ID         int64
	BuildID    int64
	VariableID sql.NullInt64
	Key        string
	Value      string

	Build    *Build
	Variable *variable.Variable
}

var _ database.Model = (*Variable)(nil)

func (v *Variable) Dest() []interface{} {
	return []interface{}{
		&v.ID,
		&v.BuildID,
		&v.VariableID,
		&v.Key,
		&v.Value,
	}
}

func (v *Variable) Bind(m database.Model) {
	switch v2 := m.(type) {
	case *Build:
		if v.BuildID == v2.ID {
			v.Build = v2
		}
	case *variable.Variable:
		if v.VariableID.Int64 == v2.ID {
			v.Variable = v2
		}
	}
}

func (v *Variable) JSON(addr string) map[string]interface{} {
	if v == nil {
		return nil
	}

	json := map[string]interface{}{
		"id":       v.ID,
		"build_id": v.BuildID,
		"key":      v.Key,
		"value":    v.Value,
	}

	if v.Build != nil {
		json["build"] = v.Build.JSON(addr)
	}

	if v.Variable != nil {
		json["variable_url"] = addr + v.Variable.Endpoint()
	}
	return json
}

func (v *Variable) Endpoint(_ ...string) string { return "" }

func (v Variable) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":          v.ID,
		"build_id":    v.BuildID,
		"variable_id": v.VariableID,
		"key":         v.Key,
		"value":       v.Value,
	}
}

type VariableStore struct {
	database.Pool
}

var (
	_ database.Loader = (*VariableStore)(nil)

	variableTable = "build_variables"
)

func (s VariableStore) All(opts ...query.Option) ([]*Variable, error) {
	vv := make([]*Variable, 0)

	new := func() database.Model {
		v := &Variable{}
		vv = append(vv, v)
		return v
	}

	if err := s.Pool.All(variableTable, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return vv, nil
}

func (s VariableStore) Load(fk, pk string, mm ...database.Model) error {
	vals := database.Values(fk, mm)

	vv, err := s.All(query.Where(pk, "IN", database.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	for _, v := range vv {
		for _, m := range mm {
			m.Bind(v)
		}
	}
	return nil
}
