// Package build providers types and functions for working with Builds and the
// related Build models.
package build

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/auth"
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
	"djinn-ci.com/variable"

	"github.com/andrewpillar/query"

	"github.com/mcmathja/curlyq"

	"github.com/go-redis/redis"
)

// Build represents a Build that has been submitted to be run.
type Build struct {
	loaded []string

	ID          int64
	UserID      int64
	NamespaceID database.Null[int64]
	Number      int64
	Manifest    manifest.Manifest
	Status      runner.Status
	Output      database.Null[string]
	Secret      database.Null[string]
	Pinned      bool
	CreatedAt   time.Time
	StartedAt   database.Null[time.Time]
	FinishedAt  database.Null[time.Time]

	User      *auth.User
	Namespace *namespace.Namespace
	Driver    *Driver
	Trigger   *Trigger
	Tags      []*Tag
	Stages    []*Stage
}

func LoadRelations(ctx context.Context, pool *database.Pool, bb ...*Build) error {
	if len(bb) == 0 {
		return nil
	}

	if err := namespace.LoadResourceRelations[*Build](ctx, pool, bb...); err != nil {
		return errors.Err(err)
	}

	rels := []database.Relation{
		{
			From: "id",
			To:   "build_id",
			Loader: database.ModelLoader(pool, tagTable, func() database.Model {
				return &Tag{}
			}),
		}, {
			From: "id",
			To:   "build_id",
			Loader: database.ModelLoader(pool, triggerTable, func() database.Model {
				return &Trigger{}
			}),
		}, {
			From: "id",
			To:   "build_id",
			Loader: database.ModelLoader(pool, stageTable, func() database.Model {
				return &Stage{}
			}),
		}, {
			From: "id",
			To:   "build_id",
			Loader: database.ModelLoader(pool, driverTable, func() database.Model {
				return &Driver{}
			}),
		}, {
			From: "id",
			To:   "build_id",
			Loader: database.ModelLoader(pool, tagTable, func() database.Model {
				return &Tag{}
			}),
		},
	}

	if err := database.LoadRelations[*Build](ctx, bb, rels...); err != nil {
		return errors.Err(err)
	}
	return nil
}

var _ database.Model = (*Build)(nil)

func (b *Build) Tag(ctx context.Context, pool *database.Pool, u *auth.User, tags ...string) error {
	if len(tags) == 0 {
		return nil
	}

	set := make(map[string]struct{})
	vals := make([]any, 0, len(tags))

	for _, tag := range tags {
		set[tag] = struct{}{}
	}
	tags = tags[0:0]

	// Remove duplicate tags.
	for tag := range set {
		tags = append(tags, tag)
		vals = append(vals, tag)

		delete(set, tag)
	}

	store := NewTagStore(pool)

	tt, err := store.Select(
		ctx,
		[]string{"name"},
		query.Where("build_id", "=", query.Arg(b.ID)),
		query.Where("name", "IN", query.List(vals...)),
	)

	if err != nil {
		return errors.Err(err)
	}

	for _, t := range tt {
		set[t.Name] = struct{}{}
	}
	tt = nil

	b.Tags = make([]*Tag, 0, len(set))

	now := time.Now()

	tx, err := pool.Begin(ctx)

	if err != nil {
		return errors.Err(err)
	}

	defer tx.Rollback(ctx)

	for _, name := range tags {
		// Ignore duplicate tags we already have.
		if _, ok := set[name]; ok {
			continue
		}

		if name == "" {
			continue
		}

		tag := &Tag{
			UserID:    u.ID,
			BuildID:   b.ID,
			Name:      name,
			CreatedAt: now,
			User:      u,
			Build:     b,
		}

		if err := store.Create(ctx, tag); err != nil {
			return errors.Err(err)
		}
		b.Tags = append(b.Tags, tag)
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.Err(err)
	}
	return nil
}

// Kill publishes the current Build's secret to the given Redis client to signal
// that the worker should immediately kill the Build's execution. This will only
// publish the secret if the Build is running.
func (b *Build) Kill(redis *redis.Client) error {
	if b.Status != runner.Running {
		println("build not running, not killing")
		return nil
	}

	key := "kill-" + strconv.FormatInt(b.ID, 10)

	if _, err := redis.Publish(key, b.Secret.Elem).Result(); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (b *Build) togglePin(ctx context.Context, pool *database.Pool) error {
	b.Pinned = !b.Pinned

	if err := NewStore(pool).Update(ctx, b); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (b *Build) Pin(ctx context.Context, pool *database.Pool) error {
	if b.Pinned {
		return nil
	}

	if err := b.togglePin(ctx, pool); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (b *Build) Unpin(ctx context.Context, pool *database.Pool) error {
	if !b.Pinned {
		return nil
	}

	if err := b.togglePin(ctx, pool); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (b *Build) Primary() (string, any) { return "id", b.ID }

func (b *Build) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":           &b.ID,
		"user_id":      &b.UserID,
		"namespace_id": &b.NamespaceID,
		"number":       &b.Number,
		"manifest":     &b.Manifest,
		"status":       &b.Status,
		"output":       &b.Output,
		"secret":       &b.Secret,
		"pinned":       &b.Pinned,
		"created_at":   &b.CreatedAt,
		"started_at":   &b.StartedAt,
		"finished_at":  &b.FinishedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}

	b.loaded = r.Columns
	return nil
}

func (b *Build) Params() database.Params {
	params := database.Params{
		"id":           database.ImmutableParam(b.ID),
		"user_id":      database.CreateOnlyParam(b.UserID),
		"namespace_id": database.CreateOnlyParam(b.NamespaceID),
		"number":       database.CreateOnlyParam(b.Number),
		"manifest":     database.CreateOnlyParam(b.Manifest),
		"status":       database.CreateUpdateParam(b.Status),
		"output":       database.CreateUpdateParam(b.Output),
		"secret":       database.CreateOnlyParam(b.Secret),
		"pinned":       database.UpdateOnlyParam(b.Pinned),
		"created_at":   database.CreateOnlyParam(b.CreatedAt),
		"started_at":   database.UpdateOnlyParam(b.StartedAt),
		"finished_at":  database.UpdateOnlyParam(b.FinishedAt),
	}

	if len(b.loaded) > 0 {
		params.Only(b.loaded...)
	}
	return params
}

// Bind the given Model to the current Build if it is one of User, Namespace,
// Trigger, Tag, Stage, or Driver, and if there is a direct relation between
// the two.
func (b *Build) Bind(m database.Model) {
	switch v := m.(type) {
	case *auth.User:
		if b.UserID == v.ID {
			b.User = v
		}
	case *namespace.Namespace:
		if b.NamespaceID.Elem == v.ID {
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

func (b *Build) MarshalJSON() ([]byte, error) {
	if b == nil {
		return []byte("null"), nil
	}

	tags := make([]string, 0, len(b.Tags))

	for _, t := range b.Tags {
		tags = append(tags, t.Name)
	}

	sort.Strings(tags)

	p, err := json.Marshal(map[string]any{
		"id":            b.ID,
		"user_id":       b.UserID,
		"namespace_id":  b.NamespaceID,
		"number":        b.Number,
		"manifest":      b.Manifest.String(),
		"status":        b.Status,
		"output":        b.Output,
		"tags":          tags,
		"pinned":        b.Pinned,
		"created_at":    b.CreatedAt,
		"started_at":    b.StartedAt,
		"finished_at":   b.FinishedAt,
		"url":           env.DJINN_API_SERVER + b.Endpoint(),
		"objects_url":   env.DJINN_API_SERVER + b.Endpoint("objects"),
		"variables_url": env.DJINN_API_SERVER + b.Endpoint("variables"),
		"jobs_url":      env.DJINN_API_SERVER + b.Endpoint("jobs"),
		"artifacts_url": env.DJINN_API_SERVER + b.Endpoint("artifacts"),
		"tags_url":      env.DJINN_API_SERVER + b.Endpoint("tags"),
		"trigger":       b.Trigger,
		"user":          b.User,
		"namespace":     b.Namespace,
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return p, nil
}

// Endpoint returns the endpoint for the current Build. this will only return
// an endpoint if the current Build has a non-nil User. The given uris are
// appended to the returned endpoint.
func (b *Build) Endpoint(elems ...string) string {
	if b.User == nil {
		return ""
	}

	endpoint := fmt.Sprintf("/b/%s/%v", b.User.Username, b.Number)

	if len(elems) > 0 {
		return fmt.Sprintf("%s/%s", endpoint, strings.Join(elems, "/"))
	}
	return endpoint
}

// Event represents an event that has been emitted for the Build. This would
// typically be a webhook event, such as for when a Build is submitted.
type Event struct {
	dis event.Dispatcher

	Build *Build
}

var (
	_ queue.Job = (*Event)(nil)
	_ queue.Job = (*PinEvent)(nil)
)

// InitEvent returns a queue.InitFunc that will initialize the queue Job with
// the given event Dispatcher if that queue Job is for a build Event.
func InitEvent(dis event.Dispatcher) queue.InitFunc {
	return func(j queue.Job) {
		switch v := j.(type) {
		case *Event:
			v.dis = dis
		case *PinEvent:
			v.dis = dis
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

	b, err := e.Build.MarshalJSON()

	if err != nil {
		return errors.Err(err)
	}

	data := make(map[string]any)
	json.Unmarshal(b, &data)

	ev := event.New(e.Build.NamespaceID, typ, data)

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

type PinEvent struct {
	dis event.Dispatcher

	Build *Build
}

func (e *PinEvent) Name() string {
	if e.Build.Pinned {
		return "event:" + event.BuildPinned.String()
	}
	return "event:" + event.BuildUnpinned.String()
}

func (e *PinEvent) Perform() error {
	if e.dis == nil {
		return event.ErrNilDispatcher
	}

	typ := event.BuildPinned

	if !e.Build.Pinned {
		typ = event.BuildUnpinned
	}

	b, err := e.Build.MarshalJSON()

	if err != nil {
		return errors.Err(err)
	}

	data := make(map[string]any)
	json.Unmarshal(b, &data)

	ev := event.New(e.Build.NamespaceID, typ, data)

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

var _ queue.Job = (*Event)(nil)

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

// WherePinned returns a query option for getting Builds that are pinned.
func WherePinned(pinned bool) query.Option {
	return func(q query.Query) query.Query {
		if !pinned {
			return q
		}
		return query.Where("pinned", "=", query.Arg(true))(q)
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

// Store allows for the management of Builds. This includes their creation,
// submission, and updating of during runs.
type Store struct {
	*database.Store[*Build]

	Hasher *crypto.Hasher
	Queues map[string]*curlyq.Producer
}

const table = "builds"

func NewStore(pool *database.Pool) *database.Store[*Build] {
	return database.NewStore[*Build](pool, table, func() *Build {
		return &Build{}
	})
}

// Index returns the paginated Builds with the given query options applied. The
// given url.Values are used to apply WhereTag, WhereSearch, and WhereStatus
// query options if the tag, search, and status values are present respectively
// in the underlying map.
func (s *Store) Index(ctx context.Context, vals url.Values, opts ...query.Option) (*database.Paginator[*Build], error) {
	page, _ := strconv.Atoi(vals.Get("page"))

	opts = append(opts,
		WhereTag(vals.Get("tag")),
		WhereSearch(vals.Get("search")),
		WhereStatus(vals.Get("status")),
		WherePinned(vals.Has("pinned")),
	)

	paginator, err := s.Paginate(ctx, page, database.PageLimit, opts...)

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := paginator.Load(ctx, s.Store, append(opts, query.OrderAsc("created_at"))...); err != nil {
		return nil, errors.Err(err)
	}
	return paginator, nil
}

type Params struct {
	User     *auth.User
	Trigger  *Trigger
	Manifest manifest.Manifest
	Tags     []string
}

func (s *Store) Create(ctx context.Context, p *Params) (*Build, error) {
	typ := p.Manifest.Driver["type"]

	if typ == "qemu" {
		p.Manifest.Driver["arch"] = "x86_64"
		typ += "-" + p.Manifest.Driver["arch"]
	}

	if _, ok := s.Queues[typ]; !ok {
		if driver.IsValid(typ) {
			return nil, driver.ErrDisabled(typ)
		}
		return nil, driver.ErrUnknown(typ)
	}

	b := Build{
		UserID:   p.User.ID,
		Manifest: p.Manifest,
		User:     p.User,
	}

	if p.Manifest.Namespace != "" {
		path, err := namespace.ParsePath(p.Manifest.Namespace)

		if err != nil {
			return nil, errors.Err(err)
		}

		u, n, err := path.Resolve(ctx, s.Pool, p.User)

		if err != nil {
			return nil, errors.Err(err)
		}

		n.User = u

		b.UserID = u.ID
		b.User = u
		b.NamespaceID = database.Null[int64]{
			Elem:  n.ID,
			Valid: true,
		}
		b.Namespace = n
	}

	last, ok, err := s.SelectOne(
		ctx,
		[]string{"number"},
		query.Where("user_id", "=", query.Arg(b.UserID)),
		query.OrderDesc("created_at"),
	)

	if err != nil {
		return nil, errors.Err(err)
	}

	if !ok {
		last = &Build{}
	}

	secret := make([]byte, 16)

	if _, err := rand.Read(secret); err != nil {
		return nil, errors.Err(err)
	}

	b.Number = last.Number + 1
	b.Status = runner.Queued
	b.Secret = database.Null[string]{
		Elem:  hex.EncodeToString(secret),
		Valid: true,
	}
	b.CreatedAt = time.Now()

	if err := s.Store.Create(ctx, &b); err != nil {
		return nil, errors.Err(err)
	}

	p.Trigger.BuildID = b.ID
	p.Trigger.CreatedAt = b.CreatedAt
	p.Trigger.Build = &b

	if err := NewTriggerStore(s.Pool).Create(ctx, p.Trigger); err != nil {
		return nil, errors.Err(err)
	}

	if err := b.Tag(ctx, s.Pool, p.User, p.Tags...); err != nil {
		return nil, errors.Err(err)
	}
	return &b, nil
}

func (s *Store) Started(ctx context.Context, b *Build) error {
	b.StartedAt = database.Null[time.Time]{
		Elem:  time.Now(),
		Valid: true,
	}

	if err := s.Update(ctx, b); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *Store) Finished(ctx context.Context, b *Build) error {
	b.FinishedAt = database.Null[time.Time]{
		Elem:  time.Now(),
		Valid: true,
	}

	b.loaded = append(b.loaded, "output", "status", "finished_at")

	if err := s.Update(ctx, b); err != nil {
		return errors.Err(err)
	}
	return nil
}

// Payload is how the Build is put onto the queue.
type Payload struct {
	Host    string // Host from which the Build was submitted.
	BuildID int64  // ID of the Build.
}

var (
	reSlug = regexp.MustCompile("[^a-zA-Z0-9.]")
	reDup  = regexp.MustCompile("-{2,}")
)

func (s *Store) Submit(ctx context.Context, host string, b *Build) error {
	tx, err := s.Begin(ctx)

	if err != nil {
		return errors.Err(err)
	}

	defer tx.Rollback(ctx)

	typ := b.Manifest.Driver["type"]

	if typ == "qemu" {
		b.Manifest.Driver["arch"] = "x86_64"
		typ += "-" + b.Manifest.Driver["arch"]
	}

	driverType, err := driver.Lookup(typ)

	if err != nil {
		return errors.Err(err)
	}

	d := Driver{
		BuildID: b.ID,
		Type:    driverType,
		Config:  b.Manifest.Driver,
	}

	if err := NewDriverStore(s.Pool).CreateTx(ctx, tx, &d); err != nil {
		return errors.Err(err)
	}

	opts := []query.Option{
		query.Where("user_id", "=", query.Arg(b.UserID)),
		query.Where("namespace_id", "IS", query.Lit("NULL")),
		query.OrWhere("namespace_id", "=", query.Arg(b.NamespaceID)),
	}

	vv, err := variable.NewStore(s.Pool).All(ctx, opts...)

	if err != nil {
		return errors.Err(err)
	}

	for _, env := range b.Manifest.Env {
		parts := strings.SplitN(env, "=", 2)

		vv = append(vv, &variable.Variable{
			Key:   parts[0],
			Value: parts[1],
		})
	}

	variables := NewVariableStore(s.Pool)

	for _, v := range vv {
		err := variables.CreateTx(ctx, tx, &Variable{
			BuildID: b.ID,
			VariableID: database.Null[int64]{
				Elem:  v.ID,
				Valid: v.ID > 0,
			},
			Key:   v.Key,
			Value: v.Value,
		})

		if err != nil {
			return errors.Err(err)
		}
	}

	kk, err := key.NewStore(s.Pool).All(ctx, opts...)

	if err != nil {
		return errors.Err(err)
	}

	keys := NewKeyStore(s.Pool)

	for _, k := range kk {
		err := keys.CreateTx(ctx, tx, &Key{
			BuildID: b.ID,
			KeyID: database.Null[int64]{
				Elem:  k.ID,
				Valid: true,
			},
			Key:      k.Key,
			Name:     k.Name,
			Config:   k.Config,
			Location: "/root/.ssh/" + k.Name,
		})

		if err != nil {
			return errors.Err(err)
		}
	}

	objnames := make([]any, 0, len(b.Manifest.Objects))

	for name := range b.Manifest.Objects {
		objnames = append(objnames, name)
	}

	if len(objnames) > 0 {
		opts = append([]query.Option{
			query.Where("name", "IN", query.List(objnames...)),
		}, opts...)

		oo, err := object.NewStore(s.Pool).All(ctx, opts...)

		if err != nil {
			return errors.Err(err)
		}

		objects := NewObjectStore(s.Pool)

		for _, o := range oo {
			err := objects.CreateTx(ctx, tx, &Object{
				BuildID: b.ID,
				ObjectID: database.Null[int64]{
					Elem:  o.ID,
					Valid: o.ID > 0,
				},
				Source: b.Manifest.Objects[o.Name],
				Name:   o.Name,
			})

			if err != nil {
				return errors.Err(err)
			}
		}
	}

	setup := "setup - #" + strconv.FormatInt(b.Number, 10)

	b.Manifest.Stages = append([]string{setup}, b.Manifest.Stages...)

	failtab := make(map[string]struct{})

	for _, stage := range b.Manifest.AllowFailures {
		failtab[stage] = struct{}{}
	}

	stagetab := make(map[string]int64)

	stages := NewStageStore(s.Pool)

	for _, name := range b.Manifest.Stages {
		_, ok := failtab[name]

		s := Stage{
			BuildID:   b.ID,
			Name:      name,
			CanFail:   ok,
			CreatedAt: time.Now(),
		}

		if err := stages.CreateTx(ctx, tx, &s); err != nil {
			return errors.Err(err)
		}
		stagetab[name] = s.ID
	}

	setupJobs := make([]manifest.Job, 0, len(b.Manifest.Sources)+1)
	setupJobs = append(setupJobs, manifest.Job{
		Stage: setup,
		Name:  "create-driver",
	})

	for i, src := range b.Manifest.Sources {
		setupJobs = append(setupJobs, manifest.Job{
			Stage: setup,
			Name:  "clone." + strconv.Itoa(i),
			Commands: []string{
				"mkdir -p " + src.Dir,
				"git clone -q " + src.URL + " " + src.Dir,
				"cd " + src.Dir,
				"git checkout " + src.Ref,
			},
		})
	}

	jobs := NewJobStore(s.Pool)

	for _, job := range setupJobs {
		err := jobs.CreateTx(ctx, tx, &Job{
			BuildID:   b.ID,
			StageID:   stagetab[job.Stage],
			Name:      job.Name,
			Commands:  strings.Join(job.Commands, "\n"),
			CreatedAt: time.Now(),
		})

		if err != nil {
			return errors.Err(err)
		}
	}

	stage := ""
	number := 1

	artifacts := &ArtifactStore{
		Store:  NewArtifactStore(s.Pool),
		Hasher: s.Hasher,
	}

	for _, job := range b.Manifest.Jobs {
		stageId, ok := stagetab[job.Stage]

		if !ok {
			continue
		}

		if job.Stage != stage {
			stage = job.Stage
			number = 1
		} else {
			number++
		}

		if job.Name == "" {
			job.Name = job.Stage + "." + strconv.Itoa(number)
		}

		// Slugify the job name.
		job.Name = strings.TrimSpace(job.Name)
		job.Name = reSlug.ReplaceAllString(job.Name, "-")
		job.Name = reDup.ReplaceAllString(job.Name, "-")
		job.Name = strings.ToLower(strings.TrimPrefix(strings.TrimSuffix(job.Name, "-"), "-"))

		j := Job{
			BuildID:   b.ID,
			StageID:   stageId,
			Name:      job.Name,
			Commands:  strings.Join(job.Commands, "\n"),
			CreatedAt: time.Now(),
		}

		if err := jobs.CreateTx(ctx, tx, &j); err != nil {
			return errors.Err(err)
		}

		if len(job.Artifacts) > 0 {
			for src, dst := range job.Artifacts {
				err := artifacts.CreateTx(ctx, tx, &Artifact{
					UserID:    b.UserID,
					BuildID:   b.ID,
					JobID:     j.ID,
					Source:    src,
					Name:      dst,
					CreatedAt: time.Now(),
				})

				if err != nil {
					return errors.Err(err)
				}
			}
		}
	}

	var buf bytes.Buffer

	gob.NewEncoder(&buf).Encode(Payload{
		Host:    host,
		BuildID: b.ID,
	})

	q, ok := s.Queues[typ]

	if !ok {
		if driver.IsValid(typ) {
			return driver.ErrDisabled(typ)
		}
		return driver.ErrUnknown(typ)
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.Err(err)
	}

	if _, err := q.PerformCtx(ctx, curlyq.Job{Data: buf.Bytes()}); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *Store) Orphan(ctx context.Context, b *Build) error {
	if err := b.Tag(ctx, s.Pool, b.User, "orphaned"); err != nil {
		return errors.Err(err)
	}

	j := Job{
		Status: runner.Queued,
	}

	if err := NewJobStore(s.Pool).UpdateMany(ctx, &j, query.Where("build_id", "=", query.Arg(b.ID))); err != nil {
		return errors.Err(err)
	}

	b.Status = runner.Queued
	b.Output = database.Null[string]{}
	b.StartedAt = database.Null[time.Time]{}
	b.FinishedAt = database.Null[time.Time]{}

	if err := s.Update(ctx, b); err != nil {
		return errors.Err(err)
	}
	return nil
}
