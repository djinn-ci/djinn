package build

import (
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"

	"github.com/andrewpillar/query"
)

type Stage struct {
	ID        int64
	BuildID   int64
	Name      string
	CanFail   bool
	CreatedAt time.Time

	Jobs []*Job
}

var _ database.Model = (*Stage)(nil)

func (s *Stage) Dest() []interface{} {
	return []interface{}{
		&s.ID,
		&s.BuildID,
		&s.Name,
		&s.CanFail,
		&s.CreatedAt,
	}
}

func (s *Stage) Bind(m database.Model) {
	if v, ok := m.(*Job); ok {
		if s.ID == v.StageID {
			s.Jobs = append(s.Jobs, v)
		}
	}
}

func (*Stage) JSON(_ string) map[string]interface{} { return nil }

func (*Stage) Endpoint(_ ...string) string { return "" }

func (s *Stage) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":       s.ID,
		"build_id": s.BuildID,
		"name":     s.Name,
		"can_fail": s.CanFail,
	}
}

type StageStore struct {
	database.Pool
}

var (
	_ database.Loader = (*StageStore)(nil)

	stageTable = "build_stages"
)

func (s StageStore) Get(opts ...query.Option) (*Stage, bool, error) {
	var st Stage

	ok, err := s.Pool.Get(stageTable, &st, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &st, ok, nil
}

func (s StageStore) All(opts ...query.Option) ([]*Stage, error) {
	ss := make([]*Stage, 0)

	new := func() database.Model {
		st := &Stage{}
		ss = append(ss, st)
		return st
	}

	if err := s.Pool.All(stageTable, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return ss, nil
}

func (s StageStore) Load(fk, pk string, mm ...database.Model) error {
	vals := database.Values(fk, mm)

	uu, err := s.All(query.Where(pk, "IN", database.List(vals...)), query.OrderAsc("created_at"))

	if err != nil {
		return errors.Err(err)
	}

	for _, u := range uu {
		for _, m := range mm {
			m.Bind(u)
		}
	}
	return nil
}
