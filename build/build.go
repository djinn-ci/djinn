// Package build provides database implementations for the Build entity and its
// related entities.
package build

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/key"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/object"
	"github.com/andrewpillar/djinn/runner"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/variable"

	"github.com/andrewpillar/query"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"

	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/tasks"
)

// Build is the type that represents a build that has either run, or will be
// run.
type Build struct {
	ID          int64           `db:"id"`
	UserID      int64           `db:"user_id"`
	NamespaceID sql.NullInt64   `db:"namespace_id"`
	Manifest    config.Manifest `db:"manifest"`
	Status      runner.Status   `db:"status"`
	Output      sql.NullString  `db:"output"`
	Secret      sql.NullString  `db:"secret"`
	CreatedAt   time.Time       `db:"created_at"`
	StartedAt   pq.NullTime     `db:"started_at"`
	FinishedAt  pq.NullTime     `db:"finished_at"`

	User      *user.User           `db:"-" json:"-"`
	Namespace *namespace.Namespace `db:"-" json:"-"`
	Driver    *Driver              `db:"-" json:"-"`
	Trigger   *Trigger             `db:"-" json:"-"`
	Tags      []*Tag               `db:"-" json:"-"`
	Stages    []*Stage             `db:"-" json:"-"`
}

// Store is the type for creating and modifying Build models in the database.
// The Store type can have an underlying hasher.Hasher that is used for
// generating artifact hashes.
type Store struct {
	database.Store

	hasher *crypto.Hasher

	// User is the bound User model. If not nil this will bind the User model to
	// any Build models that are created. If not nil this will be passed to the
	// namespace.WhereCollaborator query option on each SELECT query performed.
	User *user.User

	// Namespace is the bound Namespace model. If not nil this will bind the
	// Namespace model to any Build models that are created. If not nil this
	// will append a WHERE clause on the namespace_id column for all SELECT
	// queries performed.
	Namespace *namespace.Namespace
}

var (
	_ database.Model  = (*Build)(nil)
	_ database.Binder = (*Store)(nil)
	_ database.Loader = (*Store)(nil)

	table     = "builds"
	relations = map[string]database.RelationFunc{
		"namespace":     database.Relation("namespace_id", "id"),
		"user":          database.Relation("user_id", "id"),
		"build_tag":     database.Relation("id", "build_id"),
		"build_trigger": database.Relation("id", "build_id"),
		"build_stage":   database.Relation("id", "build_id"),
	}

	ErrDriver = errors.New("unknown driver")
)

// NewStore returns a new Store for querying the builds table. Each model
// passed to this function will be bound to the returned Store.
func NewStore(db *sqlx.DB, mm ...database.Model) *Store {
	s := &Store{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// NewStoreWithHasher functionally does the same as NewStore, however it sets
// the given hasher on the newly returned store for hashing of Artifact names.
func NewStoreWithHasher(db *sqlx.DB, hasher *crypto.Hasher, mm ...database.Model) *Store {
	s := NewStore(db, mm...)
	s.hasher = hasher
	return s
}

// FromContext returns the Build model from the given context, if any.
func FromContext(ctx context.Context) (*Build, bool) {
	b, ok := ctx.Value("build").(*Build)
	return b, ok
}

// LoadRelations loads all of the available relations for the given Build
// models using the given loaders available.
func LoadRelations(loaders *database.Loaders, bb ...*Build) error {
	mm := database.ModelSlice(len(bb), Model(bb))
	return database.LoadRelations(relations, loaders, mm...)
}

// Model is called along with database.ModelSlice to convert the given slice of
// Build models to a slice of database.Model interfaces.
func Model(bb []*Build) func(int) database.Model {
	return func(i int) database.Model {
		return bb[i]
	}
}

// WhereSearch returns a query option that applies a WHERE IN clause to a
// query. The returned WHERE IN clause operates on the values returned from a
// SELECT query that uses the given search string.
func WhereSearch(search string) query.Option {
	return func(q query.Query) query.Query {
		if search == "" {
			return q
		}
		return query.WhereQuery("id", "IN",
			query.Select(
				query.Columns("build_id"),
				query.From(tagTable),
				query.Where("name", "LIKE", "%"+search+"%"),
			),
		)(q)
	}
}

// WhereStatus returns a query option that applies a WHERE IN clause to a query
// using the given status. If the given status is "passed", then it will be
// expanded to include "passed_with_failures".
func WhereStatus(status string) query.Option {
	return func(q query.Query) query.Query {
		if status == "" {
			return q
		}

		vals := []interface{}{status}

		if status == "passed" {
			vals = append(vals, "passed_with_failures")
		}
		return query.Where("status", "IN", vals...)(q)
	}
}

// WhereTag returns a query option that applies a WHERE IN clause to a
// query. The returned WHERE IN claue operates on the values returnd from a
// SELECT query that uses the given tag string.
func WhereTag(tag string) query.Option {
	return func(q query.Query) query.Query {
		if tag == "" {
			return q
		}
		return query.WhereQuery("id", "IN",
			query.Select(
				query.Columns("build_id"),
				query.From(tagTable),
				query.Where("name", "=", tag),
			),
		)(q)
	}
}

// Kill kills the given build by publishing the build's secret to the given
// redis.Client.
func (b *Build) Kill(client *redis.Client) error {
	if b.Status != runner.Running {
		return nil
	}

	key := "kill-" + strconv.FormatInt(b.ID, 10)

	if _, err := client.Publish(key, b.Secret.String).Result(); err != nil {
		return errors.Err(err)
	}

	b.Status = runner.Killed
	return nil
}

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either user.User, namespace.Namespace, Trigger, Tag,
// Stage, or Driver.
func (b *Build) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			b.User = m.(*user.User)
		case *namespace.Namespace:
			b.Namespace = m.(*namespace.Namespace)
		case *Trigger:
			b.Trigger = m.(*Trigger)
		case *Tag:
			b.Tags = append(b.Tags, m.(*Tag))
		case *Stage:
			s := m.(*Stage)
			s.Build = b
			b.Stages = append(b.Stages, s)
		case *Driver:
			b.Driver = m.(*Driver)
		}
	}
}

// Endpoint implements the database.Model interface. If the current Build has
// a nil or zero value User bound model then an empty string is returned,
// otherwise the full Build endpoint is returned, for example,
//
//   /b/l.belardo/10
func (b *Build) Endpoint(uri ...string) string {
	if b.User == nil || b.User.IsZero() {
		return ""
	}

	endpoint := fmt.Sprintf("/b/%s/%v", b.User.Username, b.ID)

	if len(uri) > 0 {
		return fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}
	return endpoint
}

// Primary implements the database.Model interface.
func (b *Build) Primary() (string, int64) { return "id", b.ID }

// SetPrimary implements the database.Model interface.
func (b *Build) SetPrimary(id int64) { b.ID = id }

// IsZero implements the database.Model interface.
func (b *Build) IsZero() bool {
	return b == nil || b.ID == 0 &&
		!b.NamespaceID.Valid &&
		b.Manifest.String() == "" &&
		b.Status == runner.Status(0) &&
		!b.Output.Valid &&
		b.CreatedAt == time.Time{} &&
		!b.StartedAt.Valid &&
		!b.FinishedAt.Valid
}

// JSON implements the database.Model interface. This will return a map with the
// current Build values under each key. This will also include urls to the
// Build's objects, variables, jobs, artifacts, and tags. If any of the User,
// Namespace, or Trigger bound models exist on the Build, then the JSON
// representation of these models will be in the returned map, under the user,
// namespace, and trigger keys respectively.
func (b *Build) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":            b.ID,
		"user_id":       b.UserID,
		"namespace_id":  nil,
		"manifest":      b.Manifest.String(),
		"status":        b.Status.String(),
		"output":        nil,
		"created_at":    b.CreatedAt.Format(time.RFC3339),
		"started_at":    nil,
		"finished_at":   nil,
		"url":           addr + b.Endpoint(),
		"objects_url":   addr + b.Endpoint("objects"),
		"variables_url": addr + b.Endpoint("variables"),
		"jobs_url":      addr + b.Endpoint("jobs"),
		"artifacts_url": addr + b.Endpoint("artifacts"),
		"tags_url":      addr + b.Endpoint("tags"),
	}

	if b.NamespaceID.Valid {
		json["namespace_id"] = b.NamespaceID.Int64
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

	for name, m := range map[string]database.Model{
		"user":      b.User,
		"namespace": b.Namespace,
		"trigger":   b.Trigger,
	} {
		if m != nil && !m.IsZero() {
			json[name] = m.JSON(addr)
		}
	}

	if len(b.Tags) > 0 {
		tags := make([]string, 0, len(b.Tags))

		for _, t := range b.Tags {
			tags = append(tags, t.Name)
		}
		json["tags"] = tags
	}
	return json
}

// Values implements the database.Model interface. This will return a map with
// the following values, user_id, namespace_id, manifest, status, output,
// secret, started_at, and finished_at.
func (b *Build) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":      b.UserID,
		"namespace_id": b.NamespaceID,
		"manifest":     b.Manifest,
		"status":       b.Status,
		"output":       b.Output,
		"secret":       b.Secret,
		"started_at":   b.StartedAt,
		"finished_at":  b.FinishedAt,
	}
}

// Signature returns the underlying tasks.Signature that is used when
// submitting the build to the queue server. This signature will simply
// contain the ID of the current Build. Each signature is configured to
// retry 3 times before finally failing.
func (b *Build) Signature(host string) *tasks.Signature {
	return &tasks.Signature{
		Name:       "run_build",
		RetryCount: 3,
		Args: []tasks.Arg{
			tasks.Arg{
				Type:  "int64",
				Value: b.ID,
			},
			tasks.Arg{
				Type:  "string",
				Value: host,
			},
		},
	}
}

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either user.User, namespace.Namespace, Trigger, Tag,
// Stage, or Driver.
func (s *Store) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		case *namespace.Namespace:
			s.Namespace = m.(*namespace.Namespace)
		}
	}
}

// Create takes the given config.Manifest, Trigger, and tags, and creates a new
// Build model in the database. This will also create a Trigger, and tags once
// the Build has been created in the database.
func (s *Store) Create(m config.Manifest, t *Trigger, tags ...string) (*Build, error) {
	secret := make([]byte, 16)

	if _, err := rand.Read(secret); err != nil {
		return nil, errors.Err(err)
	}

	b := s.New()
	b.Secret = sql.NullString{
		String: hex.EncodeToString(secret),
		Valid:  true,
	}
	b.Manifest = m

	if m.Namespace != "" {
		n, err := namespace.NewStore(s.DB, b.User).GetByPath(m.Namespace)

		if err != nil {
			return b, errors.Err(err)
		}

		b.Namespace = n
		b.NamespaceID = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}
	}

	if err := s.Store.Create(table, b); err != nil {
		return b, errors.Err(err)
	}

	t.BuildID = b.ID

	if err := NewTriggerStore(s.DB).Create(t); err != nil {
		return b, errors.Err(err)
	}

	tt, err := NewTagStore(s.DB, b).Create(b.UserID, tags...)

	b.Tags = tt
	return b, errors.Err(err)
}

// Started marks the build of the given id as started.
func (s *Store) Started(id int64) error {
	q := query.Update(
		query.Table(table),
		query.Set("status", runner.Running),
		query.Set("started_at", time.Now()),
		query.Where("id", "=", id),
	)

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// Finished marks the build of the given id as finished, setting the output of
// the build and status to the given values.
func (s *Store) Finished(id int64, output string, status runner.Status) error {
	q := query.Update(
		query.Table(table),
		query.Set("status", status),
		query.Set("output", output),
		query.Set("finished_at", time.Now()),
		query.Where("id", "=", id),
	)

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// Paginate returns the database.Paginator for the builds table for the given
// page.
func (s *Store) Paginate(page int64, opts ...query.Option) (database.Paginator, error) {
	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		database.Where(s.Namespace, "namespace_id"),
	}, opts...)

	paginator, err := s.Store.Paginate(table, page, 25, opts...)
	return paginator, errors.Err(err)
}

// New returns a new Build binding any non-nil models to it from the current
// Store.
func (s *Store) New() *Build {
	b := &Build{
		User:      s.User,
		Namespace: s.Namespace,
	}

	if s.User != nil {
		b.UserID = s.User.ID
	}

	if s.Namespace != nil {
		b.NamespaceID = sql.NullInt64{
			Int64: s.Namespace.ID,
			Valid: true,
		}
	}
	return b
}

// All returns a slice of Build models, applying each query.Option that is
// given.
func (s *Store) All(opts ...query.Option) ([]*Build, error) {
	bb := make([]*Build, 0)

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		database.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.All(&bb, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, b := range bb {
		b.User = s.User
		b.Namespace = s.Namespace
	}
	return bb, errors.Err(err)
}

// Index returns the paginated results from the builds table depending on the
// values that are present in url.Values. Detailed below are the values that
// are used from the given url.Values,
//
// tag    - This applies the WhereTag query.Option using the value of tag
// search - This applies the WhereSearch query.Option using the value of search
// status - This applies the WhereStatus query.Option using the value of status
func (s *Store) Index(vals url.Values, opts ...query.Option) ([]*Build, database.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		WhereTag(vals.Get("tag")),
		WhereSearch(vals.Get("search")),
		WhereStatus(vals.Get("status")),
	}, opts...)

	paginator, err := s.Paginate(page, opts...)

	if err != nil {
		return []*Build{}, paginator, errors.Err(err)
	}

	bb, err := s.All(append(
		opts,
		query.OrderDesc("created_at"),
		query.Limit(paginator.Limit),
		query.Offset(paginator.Offset),
	)...)
	return bb, paginator, errors.Err(err)
}

// Get returns a single Build model, applying each query.Option that is given.
func (s *Store) Get(opts ...query.Option) (*Build, error) {
	b := &Build{
		User:      s.User,
		Namespace: s.Namespace,
	}

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		database.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.Get(b, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return b, errors.Err(err)
}

// Load loads in a slice of Build models where the given key is in the list
// of given vals. Each database is loaded individually via a call to the given
// load callback. This method calls Store.All under the hood, so any
// bound models will impact the models being loaded.
func (s *Store) Load(key string, vals []interface{}, load database.LoaderFunc) error {
	bb, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, b := range bb {
			load(i, b)
		}
	}
	return nil
}

// Submit the given build to the givern queue server. This will also create all
// of the related entities that belong to the newly submitted build, such as
// keys, objects, variables, stages, jobs, and artifacts.
func (s *Store) Submit(srv *machinery.Server, host string, b *Build) error {
	if _, err := NewDriverStore(s.DB, b).Create(b.Manifest.Driver); err != nil {
		return errors.Err(err)
	}

	vv, err := variable.NewStore(s.DB).All(
		query.Where("user_id", "=", b.UserID),
		query.WhereRaw("namespace_id", "IS", "NULL"),
		database.OrWhere(b.Namespace, "namespace_id"),
	)

	if err != nil {
		return errors.Err(err)
	}

	variables := NewVariableStore(s.DB, b)

	if _, err := variables.Copy(vv...); err != nil {
		return errors.Err(err)
	}

	for _, env := range b.Manifest.Env {
		parts := strings.SplitN(env, "=", 2)

		if _, err := variables.Create(parts[0], parts[1]); err != nil {
			return errors.Err(err)
		}
	}

	kk, err := key.NewStore(s.DB).All(
		query.Where("user_id", "=", b.UserID),
		query.WhereRaw("namespace_id", "IS", "NULL"),
		database.OrWhere(b.Namespace, "namespace_id"),
	)

	if err != nil {
		return errors.Err(err)
	}

	if _, err := NewKeyStore(s.DB, b).Copy(kk...); err != nil {
		return errors.Err(err)
	}

	names := make([]interface{}, 0, len(b.Manifest.Objects.Values))

	for src := range b.Manifest.Objects.Values {
		names = append(names, src)
	}

	if len(names) > 0 {
		oo, err := object.NewStore(s.DB).All(
			query.Where("name", "IN", names...),
			query.Where("user_id", "=", b.UserID),
			query.WhereRaw("namespace_id", "IS", "NULL"),
			database.OrWhere(b.Namespace, "namespace_id"),
		)

		if err != nil {
			return errors.Err(err)
		}

		objects := NewObjectStore(s.DB, b)

		for _, o := range oo {
			if _, err := objects.Create(o.ID, o.Name, b.Manifest.Objects.Values[o.Name]); err != nil {
				return errors.Err(err)
			}
		}
	}

	stages := NewStageStore(s.DB, b)

	setup, err := stages.Create("setup - #"+strconv.FormatInt(b.ID, 10), false)

	if err != nil {
		return errors.Err(err)
	}

	canFail := make(map[string]struct{})

	for _, stage := range b.Manifest.AllowFailures {
		canFail[stage] = struct{}{}
	}

	set := make(map[string]*Stage)

	for _, name := range b.Manifest.Stages {
		_, ok := canFail[name]

		st, err := stages.Create(name, ok)

		if err != nil {
			return errors.Err(err)
		}
		set[st.Name] = st
	}

	jobs := NewJobStore(s.DB, b, setup)

	if _, err := jobs.Create("create driver", ""); err != nil {
		return errors.Err(err)
	}

	for i, src := range b.Manifest.Sources {
		commands := []string{
			"git clone " + src.URL + " " + src.Dir,
			"cd " + src.Dir,
			"git checkout -q " + src.Ref,
		}

		if src.Dir != "" {
			commands = append([]string{"mkdir -p " + src.Dir}, commands...)
		}

		_, err := jobs.Create(
			"clone."+strconv.FormatInt(int64(i+1), 10),
			strings.Join(commands, "\n"),
		)

		if err != nil {
			return errors.Err(err)
		}
	}

	currStage := ""
	jobNumber := int64(1)

	for _, job := range b.Manifest.Jobs {
		stage, ok := set[job.Stage]

		if !ok {
			continue
		}

		if stage.Name != currStage {
			currStage = stage.Name
			jobNumber = 1
		} else {
			jobNumber++
		}

		jobs := NewJobStore(s.DB, b, stage)
		name := job.Name

		if name == "" {
			name = stage.Name + "." + strconv.FormatInt(jobNumber, 10)
		}

		j, err := jobs.Create(name, strings.Join(job.Commands, "\n"))

		if err != nil {
			return errors.Err(err)
		}

		for src, dst := range job.Artifacts.Values {
			hash, err := s.hasher.HashNow()

			if err != nil {
				return errors.Err(err)
			}

			if _, err := NewArtifactStore(s.DB, b.User, j, b).Create(hash, src, dst); err != nil {
				return errors.Err(err)
			}
		}
	}

	_, err = srv.SendTask(b.Signature(host))
	return errors.Err(err)
}
