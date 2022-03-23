// Package build providers types and functions for working with Builds and the
// related Build models.
package build

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/driver"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/key"
	"djinn-ci.com/manifest"
	"djinn-ci.com/namespace"
	"djinn-ci.com/object"
	"djinn-ci.com/queue"
	"djinn-ci.com/runner"
	"djinn-ci.com/user"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/query"

	"github.com/jackc/pgx/v4"

	"github.com/mcmathja/curlyq"

	"github.com/go-redis/redis"
)

// Build represents a Build that has been submitted to be run.
type Build struct {
	ID          int64
	UserID      int64
	NamespaceID sql.NullInt64
	Number      int64
	Manifest    manifest.Manifest
	Status      runner.Status
	Output      sql.NullString
	Secret      sql.NullString
	CreatedAt   time.Time
	StartedAt   sql.NullTime
	FinishedAt  sql.NullTime

	User      *user.User
	Namespace *namespace.Namespace
	Driver    *Driver
	Trigger   *Trigger
	Tags      []*Tag
	Stages    []*Stage
}

var (
	_ database.Model = (*Build)(nil)

	table = "builds"
)

func Relations(db database.Pool) []database.RelationFunc {
	return []database.RelationFunc{
		database.Relation("user_id", "id", user.Store{Pool: db}),
		database.Relation("id", "build_id", TagStore{Pool: db}),
		database.Relation("id", "build_id", TriggerStore{Pool: db}),
		database.Relation("id", "build_id", StageStore{Pool: db}),
	}
}

func LoadNamespaces(db database.Pool, bb ...*Build) error {
	mm := make([]database.Model, 0, len(bb))

	for _, b := range bb {
		mm = append(mm, b)
	}

	if err := namespace.Load(db, mm...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func LoadRelations(db database.Pool, bb ...*Build) error {
	mm := make([]database.Model, 0, len(bb))

	builds := make(map[int64]*Build)

	for _, b := range bb {
		mm = append(mm, b)
		builds[b.ID] = b
	}

	if err := database.LoadRelations(mm, Relations(db)...); err != nil {
		return errors.Err(err)
	}

	mm = mm[0:0]

	stages := make(map[int64]*Stage)

	for _, b := range bb {
		for _, st := range b.Stages {
			mm = append(mm, st)
			stages[st.ID] = st
		}
	}

	vals := database.Values("id", mm)

	jobs := JobStore{Pool: db}

	jj, err := jobs.All(
		query.Where("stage_id", "IN", database.List(vals...)),
		query.OrderAsc("created_at"),
	)

	if err != nil {
		return errors.Err(err)
	}

	for _, j := range jj {
		j.Build = builds[j.BuildID]
		stages[j.StageID].Bind(j)
	}
	return nil
}

// Kill publishes the current Build's secret to the given Redis client to signal
// that the worker should immediately kill the Build's execution. This will only
// publish the secret if the Build is running.
func (b *Build) Kill(redis *redis.Client) error {
	if b.Status != runner.Running {
		return nil
	}

	key := "kill-" + strconv.FormatInt(b.ID, 10)

	if _, err := redis.Publish(key, b.Secret.String).Result(); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (b *Build) Dest() []interface{} {
	return []interface{}{
		&b.ID,
		&b.UserID,
		&b.NamespaceID,
		&b.Number,
		&b.Manifest,
		&b.Status,
		&b.Output,
		&b.Secret,
		&b.CreatedAt,
		&b.StartedAt,
		&b.FinishedAt,
	}
}

// Bind the given Model to the current Build if it is one of User, Namespace,
// Trigger, Tag, Stage, or Driver, and if there is a direct relation between
// the two.
func (b *Build) Bind(m database.Model) {
	switch v := m.(type) {
	case *user.User:
		if b.UserID == v.ID {
			b.User = v
		}
	case *namespace.Namespace:
		if b.NamespaceID.Int64 == v.ID {
			b.Namespace = v
		}
	case *Trigger:
		if b.ID == v.BuildID {
			b.Trigger = v
		}
	case *Tag:
		if b.ID == v.BuildID {
			b.Tags = append(b.Tags, v)
		}
	case *Stage:
		if b.ID == v.BuildID {
			b.Stages = append(b.Stages, v)
		}
	case *Driver:
		if b.ID == v.BuildID {
			b.Driver = v
		}
	}
}

// JSON returns a map[string]interface{} representation of the current Build.
// This will include the User, Namespace, and Trigger models if they are
// non-nil.
func (b *Build) JSON(addr string) map[string]interface{} {
	if b == nil {
		return nil
	}

	tags := make([]string, 0, len(b.Tags))

	for _, t := range b.Tags {
		tags = append(tags, t.Name)
	}

	sort.Strings(tags)

	json := map[string]interface{}{
		"id":            b.ID,
		"user_id":       b.UserID,
		"namespace_id":  nil,
		"number":        b.Number,
		"manifest":      b.Manifest.String(),
		"status":        b.Status.String(),
		"output":        nil,
		"tags":          tags,
		"created_at":    b.CreatedAt.Format(time.RFC3339),
		"started_at":    nil,
		"finished_at":   nil,
		"url":           addr + b.Endpoint(),
		"objects_url":   addr + b.Endpoint("objects"),
		"variables_url": addr + b.Endpoint("variables"),
		"jobs_url":      addr + b.Endpoint("jobs"),
		"artifacts_url": addr + b.Endpoint("artifacts"),
		"tags_url":      addr + b.Endpoint("tags"),
		"trigger":       b.Trigger.JSON(addr),
	}

	if b.User != nil {
		json["user"] = b.User.JSON(addr)
	}

	if b.NamespaceID.Valid {
		json["namespace_id"] = b.NamespaceID.Int64

		if b.Namespace != nil {
			json["namespace"] = b.Namespace.JSON(addr)
		}
	}
	if b.Output.Valid {
		json["output"] = b.Output.String
	}
	if b.StartedAt.Valid {
		json["started_at"] = b.StartedAt.Time.Format(time.RFC3339)
	}
	if b.FinishedAt.Valid {
		json["finished_at"] = b.FinishedAt.Time.Format(time.RFC3339)
	}
	return json
}

// Endpoint returns the endpoint for the current Build. this will only return
// an endpoint if the current Build has a non-nil User. The given uris are
// appended to the returned endpoint.
func (b *Build) Endpoint(uri ...string) string {
	if b.User == nil {
		return ""
	}

	endpoint := fmt.Sprintf("/b/%s/%v", b.User.Username, b.Number)

	if len(uri) > 0 {
		return fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}
	return endpoint
}

// Values returns all of the values for the current Build.
func (b *Build) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":           b.ID,
		"user_id":      b.UserID,
		"namespace_id": b.NamespaceID,
		"number":       b.Number,
		"manifest":     b.Manifest,
		"status":       b.Status,
		"output":       b.Output,
		"secret":       b.Secret,
		"created_at":   b.CreatedAt,
		"started_at":   b.StartedAt,
		"finished_at":  b.FinishedAt,
	}
}

// Event represents an event that has been emitted for the Build. This would
// typically be a webhook event, such as for when a Build is submitted.
type Event struct {
	dis event.Dispatcher

	Build *Build
}

var _ queue.Job = (*Event)(nil)

// InitEvent returns a queue.InitFunc that will initialize the queue Job with
// the given event Dispatcher if that queue Job is for a build Event.
func InitEvent(dis event.Dispatcher) queue.InitFunc {
	return func(j queue.Job) {
		if ev, ok := j.(*Event); ok {
			ev.dis = dis
		}
	}
}

// Name returns the name of the Event in the format of "event:type" where type
// is the event.Type of the current Event.
func (e *Event) Name() string {
	switch e.Build.Status {
	case runner.Queued:
		return "event:" + event.BuildSubmitted.String()
	case runner.Running:
		return "event:" + event.BuildStarted.String()
	case runner.Passed:
		fallthrough
	case runner.PassedWithFailures:
		fallthrough
	case runner.Failed:
		fallthrough
	case runner.Killed:
		fallthrough
	case runner.TimedOut:
		return "event:" + event.BuildFinished.String()
	}
	return "event:build"
}

// Perform dispatches the Event and its payload to the underlying Dispatcher.
// If not Dispatcher is set, then event.ErrNilDispatcher is returned.
func (e *Event) Perform() error {
	if e.dis == nil {
		return event.ErrNilDispatcher
	}

	var typ event.Type

	switch e.Build.Status {
	case runner.Queued:
		typ = event.BuildSubmitted
	case runner.Running:
		typ = event.BuildStarted
	case runner.Passed:
		fallthrough
	case runner.PassedWithFailures:
		fallthrough
	case runner.Failed:
		fallthrough
	case runner.Killed:
		fallthrough
	case runner.TimedOut:
		typ = event.BuildFinished
	}

	ev := event.New(e.Build.NamespaceID, typ, e.Build.JSON(env.DJINN_API_SERVER))

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

// Store allows for the management of Builds. This includes their creation,
// submission, and updating of during runs.
type Store struct {
	database.Pool

	// Hasher is used for generating hashes for the artifacts a Build may have.
	Hasher *crypto.Hasher

	// DriverQueues are the driver specified queues that a Build can be
	// submitted to. Each key in the map is the name of the respective driver.
	DriverQueues map[string]*curlyq.Producer
}

// Get returns the singlar Build that can be found with the given query options
// applied, along with whether or not one could be found.
func (s *Store) Get(opts ...query.Option) (*Build, bool, error) {
	var b Build

	ok, err := s.Pool.Get(table, &b, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &b, ok, nil
}

// All returns all of the Builds that can be found with the given query options
// applied.
func (s *Store) All(opts ...query.Option) ([]*Build, error) {
	bb := make([]*Build, 0)

	new := func() database.Model {
		b := &Build{}
		bb = append(bb, b)
		return b
	}

	if err := s.Pool.All(table, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return bb, nil
}

// WhereSearch returns a query option for searching for Builds via their tags.
func WhereSearch(search string) query.Option {
	return func(q query.Query) query.Query {
		if search == "" {
			return q
		}
		return query.Where("id", "IN",
			query.Select(
				query.Columns("build_id"),
				query.From(tagTable),
				query.Where("name", "LIKE", query.Arg("%"+search+"%")),
			),
		)(q)
	}
}

// WhereStatus returns a query option for getting Builds with the given status.
// If the given status os "passed", then the query option is expanded to include
// "passed_with_failures" too.
func WhereStatus(status string) query.Option {
	return func(q query.Query) query.Query {
		if status == "" {
			return q
		}

		vals := []interface{}{status}

		if status == "passed" {
			vals = append(vals, "passed_with_failures")
		}
		return query.Where("status", "IN", query.List(vals...))(q)
	}
}

// WhereTag returns a query option for getting Builds with the given tag. Unlike
// WhereSearch, this is strict.
func WhereTag(tag string) query.Option {
	return func(q query.Query) query.Query {
		if tag == "" {
			return q
		}
		return query.Where("id", "IN",
			query.Select(
				query.Columns("build_id"),
				query.From(tagTable),
				query.Where("name", "=", query.Arg(tag)),
			),
		)(q)
	}
}

func (s *Store) Paginate(page, limit int64, opts ...query.Option) (database.Paginator, error) {
	paginator, err := s.Pool.Paginate(table, page, limit, opts...)

	if err != nil {
		return paginator, errors.Err(err)
	}
	return paginator, nil
}

// Index returns the paginated Builds with the given query options applied. The
// given url.Values are used to apply WhereTag, WhereSearch, and WhereStatus
// query options if the tag, search, and status values are present respectively
// in the underlying map.
func (s *Store) Index(vals url.Values, opts ...query.Option) ([]*Build, database.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append(opts,
		WhereTag(vals.Get("tag")),
		WhereSearch(vals.Get("search")),
		WhereStatus(vals.Get("status")),
	)

	paginator, err := s.Paginate(page, database.PageLimit, opts...)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	bb, err := s.All(append(
		opts,
		query.OrderDesc("created_at"),
		query.Limit(paginator.Limit),
		query.Offset(paginator.Offset),
	)...)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}
	return bb, paginator, nil
}

type Params struct {
	UserID   int64
	Trigger  *Trigger
	Manifest manifest.Manifest
	Tags     []string
}

func (s *Store) Create(p Params) (*Build, error) {
	driverType := p.Manifest.Driver["type"]

	if driverType == "qemu" {
		p.Manifest.Driver["arch"] = "x86_64"
		driverType += "-" + p.Manifest.Driver["arch"]
	}

	if _, ok := s.DriverQueues[driverType]; !ok {
		if driver.IsValid(driverType) {
			return nil, driver.ErrDisabled(driverType)
		}
		return nil, driver.ErrUnknown(driverType)
	}

	buf := make([]byte, 16)

	if _, err := rand.Read(buf); err != nil {
		return nil, errors.Err(err)
	}

	userId := p.UserID

	var (
		n           *namespace.Namespace
		namespaceId sql.NullInt64
	)

	if p.Manifest.Namespace != "" {
		path, err := namespace.ParsePath(p.Manifest.Namespace)

		if err != nil {
			return nil, errors.Err(err)
		}

		u, n0, err := path.ResolveOrCreate(s.Pool, p.UserID)

		if err != nil {
			return nil, errors.Err(err)
		}

		if err := n0.IsCollaborator(s.Pool, p.UserID); err != nil {
			return nil, errors.Err(err)
		}

		userId = u.ID

		n = n0
		n.User = u

		namespaceId = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}
	}

	q := query.Select(
		query.Columns("number"),
		query.From(table),
		query.Where("user_id", "=", query.Arg(userId)),
		query.OrderDesc("created_at"),
		query.Limit(1),
	)

	var number int64

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&number); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.Err(err)
		}
	}

	number++

	status := runner.Queued
	secret := hex.EncodeToString(buf)

	now := time.Now()

	q = query.Insert(
		table,
		query.Columns("user_id", "namespace_id", "number", "manifest", "status", "secret", "created_at"),
		query.Values(userId, namespaceId, number, p.Manifest, status, secret, now),
		query.Returning("id"),
	)

	var id int64

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&id); err != nil {
		return nil, errors.Err(err)
	}

	q = query.Insert(
		triggerTable,
		query.Columns("build_id", "provider_id", "repo_id", "type", "comment", "data"),
		query.Values(id, p.Trigger.ProviderID, p.Trigger.RepoID, p.Trigger.Type, p.Trigger.Comment, p.Trigger.Data),
		query.Returning("id"),
	)

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&p.Trigger.ID); err != nil {
		return nil, errors.Err(err)
	}

	tags := TagStore{Pool: s.Pool}

	tt, err := tags.Create(TagParams{
		UserID:  p.UserID,
		BuildID: id,
		Tags:    p.Tags,
	})

	if err != nil {
		return nil, errors.Err(err)
	}

	return &Build{
		ID:          id,
		UserID:      userId,
		NamespaceID: namespaceId,
		Number:      number,
		Manifest:    p.Manifest,
		Status:      status,
		Secret: sql.NullString{
			String: secret,
			Valid:  true,
		},
		CreatedAt: now,
		Namespace: n,
		Trigger:   p.Trigger,
		Tags:      tt,
	}, nil
}

func copyvars(ctx context.Context, tx pgx.Tx, buildId int64, vv []*variable.Variable) error {
	if len(vv) == 0 {
		return nil
	}

	opts := make([]query.Option, 0, len(vv))

	for _, v := range vv {
		id := sql.NullInt64{
			Int64: v.ID,
			Valid: v.ID > 0,
		}
		opts = append(opts, query.Values(buildId, id, v.Key, v.Value, v.Masked))
	}

	q := query.Insert(variableTable, query.Columns("build_id", "variable_id", "key", "value", "masked"), opts...)

	if _, err := tx.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func copykeys(ctx context.Context, tx pgx.Tx, buildId int64, kk []*key.Key) error {
	if len(kk) == 0 {
		return nil
	}

	opts := make([]query.Option, 0, len(kk))

	for _, k := range kk {
		opts = append(opts, query.Values(buildId, k.ID, k.Key, k.Name, k.Config, "/root/.ssh/"+k.Name))
	}

	q := query.Insert(keyTable, query.Columns("build_id", "key_id", "key", "name", "config", "location"), opts...)

	if _, err := tx.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

// Payload is how the Build is put onto the queue.
type Payload struct {
	Host    string // Host from which the Build was submitted.
	BuildID int64  // ID of the Build.
}

// Submit the given Build to the underlying queue. If the Build has an invalid
// driver configured, then *driver.Error is returned.
func (s *Store) Submit(ctx context.Context, host string, b *Build) error {
	tx, err := s.Begin(ctx)

	if err != nil {
		return errors.Err(err)
	}

	defer tx.Rollback(ctx)

	driverType := b.Manifest.Driver["type"]

	// Limit to x86_64 since that's all we support right now.
	if driverType == "qemu" {
		b.Manifest.Driver["arch"] = "x86_64"
		driverType += "-" + b.Manifest.Driver["arch"]
	}

	q := query.Insert(
		driverTable,
		query.Columns("build_id", "type", "config"),
		query.Values(b.ID, b.Manifest.Driver["type"], b.Manifest.Driver),
	)

	if _, err := tx.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}

	variables := variable.Store{Pool: s.Pool}

	vv, err := variables.All(
		query.Where("user_id", "=", query.Arg(b.UserID)),
		query.Where("namespace_id", "IS", query.Lit("NULL")),
		query.OrWhere("namespace_id", "=", query.Arg(b.NamespaceID)),
	)

	if err != nil {
		return errors.Err(err)
	}

	if err := copyvars(ctx, tx, b.ID, vv); err != nil {
		return errors.Err(err)
	}

	envvars := make([]*variable.Variable, 0, len(b.Manifest.Env))

	for _, env := range b.Manifest.Env {
		parts := strings.SplitN(env, "=", 2)

		envvars = append(envvars, &variable.Variable{
			Key:   parts[0],
			Value: parts[1],
		})
	}

	if err := copyvars(ctx, tx, b.ID, envvars); err != nil {
		return errors.Err(err)
	}

	keys := key.Store{Pool: s.Pool}

	kk, err := keys.All(
		query.Where("user_id", "=", query.Arg(b.UserID)),
		query.Where("namespace_id", "IS", query.Lit("NULL")),
		query.OrWhere("namespace_id", "=", query.Arg(b.NamespaceID)),
	)

	if err != nil {
		return errors.Err(err)
	}

	if err := copykeys(ctx, tx, b.ID, kk); err != nil {
		return errors.Err(err)
	}

	names := make([]interface{}, 0, len(b.Manifest.Objects.Values))

	for src := range b.Manifest.Objects.Values {
		names = append(names, src)
	}

	if len(names) > 0 {
		objects := object.Store{Pool: s.Pool}

		oo, err := objects.All(
			query.Where("name", "IN", query.List(names...)),
			query.Where("user_id", "=", query.Arg(b.UserID)),
			query.Where("namespace_id", "IS", query.Lit("NULL")),
			query.OrWhere("namespace_id", "=", query.Arg(b.NamespaceID)),
		)

		if err != nil {
			return errors.Err(err)
		}

		opts := make([]query.Option, 0, len(oo))

		now := time.Now()

		for _, o := range oo {
			opts = append(opts, query.Values(b.ID, o.ID, b.Manifest.Objects.Values[o.Name], o.Name, now))
		}

		q := query.Insert(objectTable, query.Columns("build_id", "object_id", "source", "name", "created_at"), opts...)

		if _, err := tx.Exec(ctx, q.Build(), q.Args()...); err != nil {
			return errors.Err(err)
		}
	}

	setupStage := "setup - #" + strconv.FormatInt(b.Number, 10)

	// Prepend pseudo-stage for setting up the build environment with driver
	// creation and source cloning.
	b.Manifest.Stages = append([]string{setupStage}, b.Manifest.Stages...)

	canFail := make(map[string]struct{})

	for _, stage := range b.Manifest.AllowFailures {
		canFail[stage] = struct{}{}
	}

	var stageId int64

	stages := make(map[string]int64)

	for _, name := range b.Manifest.Stages {
		_, ok := canFail[name]

		q := query.Insert(
			stageTable,
			query.Columns("build_id", "name", "can_fail", "created_at"),
			query.Values(b.ID, name, ok, time.Now()),
			query.Returning("id"),
		)

		if err := tx.QueryRow(ctx, q.Build(), q.Args()...).Scan(&stageId); err != nil {
			return errors.Err(err)
		}
		stages[name] = stageId
	}

	setupJobs := make([]manifest.Job, 0, len(b.Manifest.Sources)+1)

	// Pseudo-job for capturing the output during driver creation.
	setupJobs = append(setupJobs, manifest.Job{
		Stage: setupStage,
		Name:  "create-driver",
	})

	for i, src := range b.Manifest.Sources {
		setupJobs = append(setupJobs, manifest.Job{
			Stage: setupStage,
			Name:  "clone." + strconv.FormatInt(int64(i+1), 10),
			Commands: []string{
				"mkdir -p " + src.Dir,
				"git clone -q " + src.URL + " " + src.Dir,
				"cd " + src.Dir,
				"git checkout " + src.Ref,
			},
		})
	}

	opts := make([]query.Option, 0, len(setupJobs))

	for _, job := range setupJobs {
		stageId, _ := stages[job.Stage]

		opts = append(opts, query.Values(b.ID, stageId, job.Name, strings.Join(job.Commands, "\n"), time.Now()))
	}

	q = query.Insert(jobTable, query.Columns("build_id", "stage_id", "name", "commands", "created_at"), opts...)

	if _, err := tx.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}

	currStage := ""
	jobNumber := int64(1)

	var jobId int64

	for _, job := range b.Manifest.Jobs {
		stageId, ok := stages[job.Stage]

		if !ok {
			continue
		}

		if job.Stage != currStage {
			currStage = job.Stage
			jobNumber = 1
		} else {
			jobNumber++
		}

		if job.Name == "" {
			job.Name = job.Stage + "." + strconv.FormatInt(jobNumber, 10)
		}

		q = query.Insert(
			jobTable,
			query.Columns("build_id", "stage_id", "name", "commands", "created_at"),
			query.Values(b.ID, stageId, job.Name, strings.Join(job.Commands, "\n"), time.Now()),
			query.Returning("id"),
		)

		if err := tx.QueryRow(ctx, q.Build(), q.Args()...).Scan(&jobId); err != nil {
			return errors.Err(err)
		}

		if len(job.Artifacts.Values) > 0 {
			opts = make([]query.Option, 0, len(job.Artifacts.Values))

			for src, dst := range job.Artifacts.Values {
				hash, err := s.Hasher.HashNow()

				if err != nil {
					return errors.Err(err)
				}

				opts = append(opts, query.Values(b.UserID, b.ID, jobId, hash, src, dst, time.Now()))
			}

			q = query.Insert(artifactTable, query.Columns("user_id", "build_id", "job_id", "hash", "source", "name", "created_at"), opts...)

			if _, err := tx.Exec(ctx, q.Build(), q.Args()...); err != nil {
				return errors.Err(err)
			}
		}
	}

	var buf bytes.Buffer

	gob.NewEncoder(&buf).Encode(Payload{
		Host:    host,
		BuildID: b.ID,
	})

	driverq, ok := s.DriverQueues[driverType]

	if !ok {
		if driver.IsValid(driverType) {
			return driver.ErrDisabled(driverType)
		}
		return driver.ErrUnknown(driverType)
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.Err(err)
	}

	if _, err := driverq.PerformCtx(ctx, curlyq.Job{Data: buf.Bytes()}); err != nil {
		return errors.Err(err)
	}
	return nil
}

// Orphan marks the given Build as orphaned and clears down the output of the
// Build and its jobs, resetting the status back to queued.
func (s *Store) Orphan(id, userId int64) error {
	tags := TagStore{Pool: s.Pool}

	params := TagParams{
		UserID:  userId,
		BuildID: id,
		Tags:    []string{"orphaned"},
	}

	if _, err := tags.Create(params); err != nil {
		return errors.Err(err)
	}

	q := query.Update(
		jobTable,
		query.Set("status", query.Arg(runner.Queued)),
		query.Set("output", query.Lit("NULL")),
		query.Set("started_at", query.Lit("NULL")),
		query.Set("finished_at", query.Lit("NULL")),
		query.Where("build_id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}

	q = query.Update(
		table,
		query.Set("status", query.Arg(runner.Queued)),
		query.Set("output", query.Lit("NULL")),
		query.Set("started_at", query.Lit("NULL")),
		query.Set("finished_at", query.Lit("NULL")),
		query.Where("id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

// Started sets the status of the Build with the given id to running.
func (s *Store) Started(id int64) error {
	q := query.Update(
		table,
		query.Set("status", query.Arg(runner.Running)),
		query.Set("started_at", query.Arg(time.Now())),
		query.Where("id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

// Finished marks the Build with the given id as finished, setting the output
// and status of the Build respectively.
func (s *Store) Finished(id int64, output string, status runner.Status) error {
	q := query.Update(
		table,
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
