// Package build provides model implementations for the Build entity and its
// related entities.
//
// Each entity implemented, has a corresponding Store which is used for
// creating, updating, and deleting models, as well as querying them from the
// database. These methods wrap the onces provided by the model package via
// model.Store.
//
// Entities:
//   - Artifact
//   - Build
//   - Driver
//   - Job
//   - Key
//   - Object
//   - Stage
//   - Tag
//   - Trigger
//   - Variable
package build

import (
	"database/sql"
	"net/url"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/runner"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"

	"github.com/RichardKnop/machinery/v1/tasks"
)

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

type Store struct {
	model.Store

	User      *user.User
	Namespace *namespace.Namespace
}

var (
	_ model.Model  = (*Build)(nil)
	_ model.Binder = (*Store)(nil)
	_ model.Loader = (*Store)(nil)

	table     = "builds"
	relations = map[string]model.RelationFunc{
		"namespace":     model.Relation("namespace_id", "id"),
		"user":          model.Relation("user_id", "id"),
		"build_tag":     model.Relation("id", "build_id"),
		"build_trigger": model.Relation("id", "build_id"),
		"build_stage":   model.Relation("id", "build_id"),
	}

	// ErrDriver denotes when an unknown driver was used for a build.
	ErrDriver = errors.New("unknown driver")
)

// NewStore returns a new Store for querying the builds table. Each model
// passed to this function will be bound to the returned Store.
func NewStore(db *sqlx.DB, mm ...model.Model) *Store {
	s := &Store{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// LoadRelations loads all of the available relations for the given Build
// models using the given loaders available.
func LoadRelations(loaders model.Loaders, bb ...*Build) error {
	mm := model.Slice(len(bb), Model(bb))
	return model.LoadRelations(relations, loaders, mm...)
}

// Model is called along with model.Slice to convert the given slice of Build
// models to a slice of model.Model interfaces.
func Model(bb []*Build) func(int) model.Model {
	return func(i int) model.Model {
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

// Bind the given models to the current Build. This will only bind the model if
// they are one of the following,
//
// - *user.User
// - *namespace.Namespace
// - *Trigger
// - *Tag
// - *Stage
// - *Driver
func (b *Build) Bind(mm ...model.Model) {
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

// Endpoint returns the endpoint for the current Build. If no User model is
// bound to the current model, then an empty string is returned. If no host is
// given then just the endpoint is returned with no host.
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

func (b *Build) Primary() (string, int64) {
	return "id", b.ID
}

func (b *Build) SetPrimary(id int64) {
	b.ID = id
}

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
		"manifest_url":  addr + b.Endpoint("manifest"),
		"output_url":    addr + b.Endpoint("output"),
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

	for name, m := range map[string]model.Model{
		"user":      b.User,
		"namespace": b.Namespace,
		"trigger":   b.Trigger,
	}{
		if !m.IsZero() {
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
func (b *Build) Signature() *tasks.Signature {
	return &tasks.Signature{
		Name:       "run_build",
		RetryCount: 3,
		Args:       []tasks.Arg{
			tasks.Arg{
				Type: "int64",
				Value: b.ID,
			},
		},
	}
}

// Bind the given models to the current Store. This will only bind the
// model if they are one of the following,
//
// - *user.User
// - *namespace.Namespace
// - *Trigger
// - *Tag
// - *Stage
// - *Driver
func (s *Store) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		case *namespace.Namespace:
			s.Namespace = m.(*namespace.Namespace)
		}
	}
}

// Create inserts the given Build models into the builds table.
func (s *Store) Create(bb ...*Build) error {
	models := model.Slice(len(bb), Model(bb))
	return errors.Err(s.Store.Create(table, models...))
}

// Update updates the given Build models in the builds table.
func (s *Store) Update(bb ...*Build) error {
	models := model.Slice(len(bb), Model(bb))
	return errors.Err(s.Store.Update(table, models...))
}

// Paginate returns the model.Paginator for the builds table for the given
// page. This applies the namespace.WhereCollaborator option to the *user.User
// bound model, and the model.Where option to the *namespace.Namespace bound
// model.
func (s *Store) Paginate(page int64, opts ...query.Option) (model.Paginator, error) {
	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	paginator, err := s.Store.Paginate(table, page, opts...)
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
// given. The namespace.WhereCollaborator option is applied to the *user.User
// bound model, and the model.Where option is applied to the
// *namespace.Namespace bound model.
func (s *Store) All(opts ...query.Option) ([]*Build, error) {
	bb := make([]*Build, 0)

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		model.Where(s.Namespace, "namespace_id"),
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
func (s *Store) Index(vals url.Values, opts ...query.Option) ([]*Build, model.Paginator, error) {
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
		query.Limit(model.PageLimit),
		query.Offset(paginator.Offset),
	)...)
	return bb, paginator, errors.Err(err)
}

// Get returns a single Build model, appling each query.Option that is given
// The namespace.WhereCollaborator option is applied to the *user.User bound
// model, and the model.Where option is applied to the *namespace.Namespace
// bound model.
func (s *Store) Get(opts ...query.Option) (*Build, error) {
	b := &Build{
		User:      s.User,
		Namespace: s.Namespace,
	}

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.Get(b, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return b, errors.Err(err)
}

// Load loads in a slice of Build models where the given key is in the list
// of given vals. Each model is loaded individually via a call to the given
// load callback. This method calls Store.All under the hood, so any
// bound models will impact the models being loaded.
func (s *Store) Load(key string, vals []interface{}, load model.LoaderFunc) error {
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
