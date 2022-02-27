package build

import (
	"context"
	"database/sql"
	"io"
	"net/url"
	"strings"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/runner"

	"github.com/andrewpillar/query"

	"github.com/jackc/pgx/v4"
)

// Job represents a single build Job.
type Job struct {
	ID         int64
	BuildID    int64
	StageID    int64
	Name       string
	Commands   string
	Status     runner.Status
	Output     sql.NullString
	CreatedAt  time.Time
	StartedAt  sql.NullTime
	FinishedAt sql.NullTime

	Build     *Build
	Stage     *Stage
	Artifacts []*Artifact
}

var _ database.Model = (*Job)(nil)

func JobRelations(db database.Pool) []database.RelationFunc {
	return []database.RelationFunc{
		database.Relation("stage_id", "id", StageStore{Pool: db}),
		database.Relation("id", "job_id", &ArtifactStore{Pool: db}),
	}
}

func LoadJobRelations(db database.Pool, jj ...*Job) error {
	mm := make([]database.Model, 0, len(jj))

	for _, j := range jj {
		mm = append(mm, j)
	}

	if err := database.LoadRelations(mm, JobRelations(db)...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (j *Job) Dest() []interface{} {
	return []interface{}{
		&j.ID,
		&j.BuildID,
		&j.StageID,
		&j.Name,
		&j.Commands,
		&j.Output,
		&j.Status,
		&j.CreatedAt,
		&j.StartedAt,
		&j.FinishedAt,
	}
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

// JSON returns a map[string]interface{} representation of the current Job.
// This will include the Build model if it is non-nil. If the Stage model
// is non-nil, then the name of the stage will be set.
func (j *Job) JSON(addr string) map[string]interface{} {
	if j == nil {
		return nil
	}

	json := map[string]interface{}{
		"id":          j.ID,
		"build_id":    j.BuildID,
		"name":        j.Name,
		"commands":    j.Commands,
		"status":      j.Status.String(),
		"output":      nil,
		"created_at":  j.CreatedAt.Format(time.RFC3339),
		"started_at":  nil,
		"finished_at": nil,
		"url":         addr + j.Endpoint(),
	}

	if j.Output.Valid {
		json["output"] = j.Output.String
	}
	if j.StartedAt.Valid {
		json["started_at"] = j.StartedAt.Time.Format(time.RFC3339)
	}
	if j.FinishedAt.Valid {
		json["finished_at"] = j.FinishedAt.Time.Format(time.RFC3339)
	}
	if j.Build != nil {
		json["build"] = j.Build.JSON(addr)
	}
	if j.Stage != nil {
		json["stage"] = j.Stage.Name
	}
	return json
}

// Endpoint returns the endpoint for the current Job. this will only return an
// endpoint if the current Job has a non-nil build. The given uris are appended
// to the returned endpoint.
func (j *Job) Endpoint(uri ...string) string {
	if j.Build == nil {
		return ""
	}
	return j.Build.Endpoint(append([]string{"jobs", j.Name}, uri...)...)
}

// Values returns all of the values for the current Job.
func (j *Job) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":          j.ID,
		"build_id":    j.BuildID,
		"stage_id":    j.StageID,
		"name":        j.Name,
		"commands":    j.Commands,
		"status":      j.Status,
		"output":      j.Output,
		"created_at":  j.CreatedAt,
		"started_at":  j.StartedAt,
		"finished_at": j.FinishedAt,
	}
}

// ArtifactStore allows for the retrieval and management of build Jobs.
type JobStore struct {
	database.Pool
}

var (
	_ database.Loader = (*JobStore)(nil)

	jobTable = "build_jobs"
)

// Job returns a runner.Job for the current Job for execution during a build.
// The given io.Writer is used for capturing the output of the Job.
func (j *Job) Job(w io.Writer) *runner.Job {
	var artifacts runner.Passthrough

	for _, a := range j.Artifacts {
		artifacts.Set(a.Source, a.Name)
	}

	return &runner.Job{
		Writer:    w,
		Name:      j.Name,
		Commands:  strings.Split(j.Commands, "\n"),
		Artifacts: artifacts,
	}
}

// Started sets the status of the Job with the given id to running.
func (s JobStore) Started(id int64) error {
	q := query.Update(
		jobTable,
		query.Set("status", query.Arg(runner.Running)),
		query.Set("started_at", query.Arg(time.Now())),
		query.Where("id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

// sanitized will sanitize any NUL bytes found in the given string by replacing
// then with the string NUL. This ensures the string is safe for UTF-8 encoding
// during database insertion.
func sanitize(s string) string {
	sanitized := make([]rune, 0, len(s))

	for _, r := range s {
		if r == 0 {
			sanitized = append(sanitized, 'N', 'U', 'L')
			continue
		}
		sanitized = append(sanitized, r)
	}
	return string(sanitized)
}

// Finished marks the Job with the given id as finished, setting the output
// and status of the Build respectively.
func (s JobStore) Finished(id int64, output string, status runner.Status) error {
	q := query.Update(
		jobTable,
		query.Set("status", query.Arg(status)),
		query.Set("output", query.Arg(sanitize(output))),
		query.Set("finished_at", query.Arg(time.Now())),
		query.Where("id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

// FinishedTx functions the same as Finished, however it performs the update
// as part of a transaction.
func (s JobStore) FinishedTx(ctx context.Context, tx pgx.Tx, id int64, output string, status runner.Status) error {
	q := query.Update(
		jobTable,
		query.Set("status", query.Arg(status)),
		query.Set("output", query.Arg(sanitize(output))),
		query.Set("finished_at", query.Arg(time.Now())),
		query.Where("id", "=", query.Arg(id)),
	)

	if _, err := tx.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

// Get returns the singular build Job that can be found with the given query
// options applied, along with whether or not one could be found.
func (s JobStore) Get(opts ...query.Option) (*Job, bool, error) {
	var j Job

	ok, err := s.Pool.Get(jobTable, &j, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &j, ok, nil
}

// All returns all of the build Jobs that can be found with the given query
// options applied.
func (s JobStore) All(opts ...query.Option) ([]*Job, error) {
	jj := make([]*Job, 0)

	new := func() database.Model {
		j := &Job{}
		jj = append(jj, j)
		return j
	}

	if err := s.Pool.All(jobTable, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return jj, nil
}

// Index returns the Jobs with the given query options applied. The given
// url.Values are used to apply the database.Search, and WhereStatus query
// options if the name, and search values are present in the underlying map.
func (s JobStore) Index(vals url.Values, opts ...query.Option) ([]*Job, error) {
	opts = append([]query.Option{
		database.Search("name", vals.Get("name")),
		WhereStatus(vals.Get("status")),
	}, opts...)

	jj, err := s.All(append(
		opts,
		query.OrderAsc("created_at"),
	)...)

	if err != nil {
		return nil, errors.Err(err)
	}
	return jj, nil
}

func (s JobStore) Load(fk, pk string, mm ...database.Model) error {
	vals := database.Values(fk, mm)

	jj, err := s.All(query.Where(pk, "IN", database.List(vals...)), query.OrderAsc("created_at"))

	if err != nil {
		return errors.Err(err)
	}

	loaded := make([]database.Model, 0, len(jj))

	for _, j := range jj {
		loaded = append(loaded, j)
	}

	database.Bind(fk, pk, loaded, mm)
	return nil
}
