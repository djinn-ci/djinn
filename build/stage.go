package build

import (
	"context"
	"encoding/json"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/runner"
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

func LoadStageRelations(ctx context.Context, db *database.Pool, ss ...*Stage) error {
	rel := database.Relation{
		From:   "id",
		To:     "stage_id",
		Loader: database.ModelLoader(db, jobTable, func() database.Model {
			return &Job{}
		}),
	}

	if err := database.LoadRelations[*Stage](ctx, ss, rel); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *Stage) Primary() (string, any) { return "id", s.ID }

func (s *Stage) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":         &s.ID,
		"build_id":   &s.BuildID,
		"name":       &s.Name,
		"can_fail":   &s.CanFail,
		"created_at": &s.CreatedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *Stage) Params() database.Params {
	return database.Params{
		"id":         database.ImmutableParam(s.ID),
		"build_id":   database.CreateOnlyParam(s.BuildID),
		"name":       database.CreateOnlyParam(s.Name),
		"can_fail":   database.CreateOnlyParam(s.CanFail),
		"created_at": database.CreateOnlyParam(s.CreatedAt),
	}
}

func (s *Stage) Bind(m database.Model) {
	if v, ok := m.(*Job); ok {
		if s.ID == v.StageID {
			s.Jobs = append(s.Jobs, v)
		}
	}
}

func (s *Stage) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	return json.Marshal(s.Name)
}

func (*Stage) Endpoint(...string) string { return "" }

func (s *Stage) Stage() *runner.Stage {
	return &runner.Stage{
		Name:    s.Name,
		CanFail: s.CanFail,
	}
}

const stageTable = "build_stages"

func NewStageStore(pool *database.Pool) *database.Store[*Stage] {
	return database.NewStore[*Stage](pool, stageTable, func() *Stage {
		return &Stage{}
	})
}
