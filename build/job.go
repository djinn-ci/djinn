package build

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/runner"

	"github.com/andrewpillar/query"
)

// Job represents a single build Job.
type Job struct {
	loaded []string

	ID         int64
	BuildID    int64
	StageID    int64
	Name       string
	Commands   string
	Status     runner.Status
	Output     database.Null[string]
	CreatedAt  time.Time
	StartedAt  database.Null[time.Time]
	FinishedAt database.Null[time.Time]

	Build     *Build
	Stage     *Stage
	Artifacts []*Artifact
}

func LoadJobRelations(ctx context.Context, pool *database.Pool, jj ...*Job) error {
	rels := []database.Relation{
		{
			From: "stage_id",
			To:   "id",
			Loader: database.ModelLoader(pool, stageTable, func() database.Model {
				return &Stage{}
			}),
		},
		{
			From: "id",
			To:   "job_id",
			Loader: database.ModelLoader(pool, artifactTable, func() database.Model {
				return &Artifact{}
			}),
		},
	}

	if err := database.LoadRelations[*Job](ctx, jj, rels...); err != nil {
		return errors.Err(err)
	}
	return nil
}

var _ database.Model = (*Job)(nil)

func (j *Job) Primary() (string, any) { return "id", j.ID }

func (j *Job) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":          &j.ID,
		"build_id":    &j.BuildID,
		"stage_id":    &j.StageID,
		"name":        &j.Name,
		"commands":    &j.Commands,
		"status":      &j.Status,
		"output":      &j.Output,
		"created_at":  &j.CreatedAt,
		"started_at":  &j.StartedAt,
		"finished_at": &j.FinishedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}

	j.loaded = r.Columns
	return nil
}

func (j *Job) Params() database.Params {
	params := database.Params{
		"id":          database.ImmutableParam(j.ID),
		"build_id":    database.CreateOnlyParam(j.BuildID),
		"stage_id":    database.CreateOnlyParam(j.StageID),
		"name":        database.CreateOnlyParam(j.Name),
		"commands":    database.CreateOnlyParam(j.Commands),
		"status":      database.CreateUpdateParam(j.Status),
		"output":      database.UpdateOnlyParam(j.Output),
		"created_at":  database.CreateOnlyParam(j.CreatedAt),
		"started_at":  database.UpdateOnlyParam(j.StartedAt),
		"finished_at": database.UpdateOnlyParam(j.FinishedAt),
	}

	if len(j.loaded) > 0 {
		params.Only(j.loaded...)
	}
	return params
}

// Bind the given Model to the current Job if it is one of Build, Stage, or
// Artifact, and if there is a direct relation between the two.
func (j *Job) Bind(m database.Model) {
	switch v := m.(type) {
	case *Build:
		if j.BuildID == v.ID {
			j.Build = v
		}
	case *Stage:
		if j.StageID == v.ID {
			j.Stage = v
		}
	case *Artifact:
		if j.ID == v.JobID {
			j.Artifacts = append(j.Artifacts, v)
		}
	}
}

func (j *Job) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}

	b, err := json.Marshal(map[string]any{
		"build_id":    j.BuildID,
		"name":        j.Name,
		"commands":    j.Commands,
		"status":      j.Status,
		"output":      j.Output,
		"created_at":  j.CreatedAt,
		"started_at":  j.StartedAt,
		"finished_at": j.FinishedAt,
		"url":         env.DJINN_API_SERVER + j.Endpoint(),
		"build":       j.Build,
		"stage":       j.Stage.Name,
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

// Endpoint returns the endpoint for the current Job. this will only return an
// endpoint if the current Job has a non-nil build. The given uris are appended
// to the returned endpoint.
func (j *Job) Endpoint(elems ...string) string {
	if j.Build == nil {
		return ""
	}
	return j.Build.Endpoint(append([]string{"jobs", j.Name}, elems...)...)
}

type JobStore struct {
	*database.Store[*Job]
}

const jobTable = "build_jobs"

func NewJobStore(pool *database.Pool) *database.Store[*Job] {
	return database.NewStore[*Job](pool, jobTable, func() *Job {
		return &Job{}
	})
}

func (s JobStore) Started(ctx context.Context, j *Job) error {
	j.StartedAt = database.Null[time.Time]{
		Elem:  time.Now(),
		Valid: true,
	}

	j.loaded = append(j.loaded, "started_at")

	if err := s.Update(ctx, j); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s JobStore) Finished(ctx context.Context, j *Job) error {
	j.FinishedAt = database.Null[time.Time]{
		Elem:  time.Now(),
		Valid: true,
	}

	if err := s.Update(ctx, j); err != nil {
		return errors.Err(err)
	}
	return nil
}

// Index returns the Jobs with the given query options applied. The given
// url.Values are used to apply the database.Search, and WhereStatus query
// options if the name, and search values are present in the underlying map.
func (s JobStore) Index(ctx context.Context, vals url.Values, opts ...query.Option) ([]*Job, error) {
	opts = append([]query.Option{
		database.Search("name", vals.Get("name")),
		WhereStatus(vals.Get("status")),
	}, opts...)

	jj, err := s.All(ctx, append(opts, query.OrderAsc("created_at"))...)

	if err != nil {
		return nil, errors.Err(err)
	}
	return jj, nil
}
