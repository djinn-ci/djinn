package build

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
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
	}

	ErrDriver = errors.New("unknown driver")
)

func NewStore(db *sqlx.DB, mm ...model.Model) Store {
	s := Store{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// LoadRelations loads all of the available relations for the given builds
// using the given loaders available.
func LoadRelations(loaders model.Loaders, bb ...*Build) error {
	mm := model.Slice(len(bb), Model(bb))
	return model.LoadRelations(relations, loaders, mm...)
}

// Model is called along with model.Slice to convert the given slice of builds
// to a slice of models.
func Model(bb []*Build) func(int) model.Model {
	return func(i int) model.Model {
		return bb[i]
	}
}

// WhereSearch returns a query option to restrain the builds returned from a
// query by the tag names that match the given search string.
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

// WhereStatus returns a query option to restrain the builds returned from a
// query by the status matching the given string.
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

// WhereTag returns a query option to restrain the builds returned from a
// query by the name of the given tag string.
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

// Bind the given models to the current Build model.
func (b *Build) Bind(mm ...model.Model) {
	if b == nil {
		return
	}

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
			b.Stages = append(b.Stages, m.(*Stage))
		case *Driver:
			b.Driver = m.(*Driver)
		}
	}
}

func (*Build) Kind() string { return "build" }

// Endpoint returns the endpoint for the current build. If nil, or if missing a
// bound User model, then an empty string is returned.
func (b *Build) Endpoint(uri ...string) string {
	if b == nil {
		return ""
	}

	if b.User == nil || b.User.IsZero() {
		return ""
	}

	username, _ := b.User.Values()["username"].(string)

	endpoint := fmt.Sprintf("/b/%s/%v", username, b.ID)

	if len(uri) > 0 {
		return fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}
	return endpoint
}

func (b *Build) Primary() (string, int64) {
	if b == nil {
		return "id", 0
	}
	return "id", b.ID
}

func (b *Build) SetPrimary(id int64) {
	if b == nil {
		return
	}
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

func (b *Build) Values() map[string]interface{} {
	if b == nil {
		return map[string]interface{}{}
	}

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

// Signature returns the signature of the task that will be submitted to the
// queue.
func (b Build) Signature() *tasks.Signature {
	buf := &bytes.Buffer{}

	enc := json.NewEncoder(buf)
	enc.Encode(b)

	return &tasks.Signature{
		Name:       "run_build",
		RetryCount: 3,
		Args:       []tasks.Arg{
			tasks.Arg{
				Type: "string",
				Value: buf.String(),
			},
		},
	}
}

// Bind the given models to the current Store.
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

func (s Store) Create(bb ...*Build) error {
	models := model.Slice(len(bb), Model(bb))
	return errors.Err(s.Store.Create(table, models...))
}

func (s Store) Update(bb ...*Build) error {
	models := model.Slice(len(bb), Model(bb))
	return errors.Err(s.Store.Update(table, models...))
}

func (s Store) Paginate(page int64, opts ...query.Option) (model.Paginator, error) {
	opts = append([]query.Option{
		model.Where(s.User, "user_id"),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	paginator, err := s.Store.Paginate(table, page, opts...)
	return paginator, errors.Err(err)
}

func (s Store) New() *Build {
	b := &Build{
		User:      s.User,
		Namespace: s.Namespace,
	}

	if s.User != nil {
		_, id := s.User.Primary()
		b.UserID = id
	}

	if s.Namespace != nil {
		_, id := s.Namespace.Primary()
		b.NamespaceID = sql.NullInt64{
			Int64: id,
			Valid: true,
		}
	}
	return b
}

func (s Store) All(opts ...query.Option) ([]*Build, error) {
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

func (s Store) Index(r *http.Request, opts ...query.Option) ([]*Build, model.Paginator, error) {
	q := r.URL.Query()

	page, err := strconv.ParseInt(q.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		WhereTag(q.Get("tag")),
		WhereSearch(q.Get("search")),
		WhereStatus(q.Get("status")),
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

func (s Store) Get(opts ...query.Option) (*Build, error) {
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

func (s Store) Load(key string, vals []interface{}, load model.LoaderFunc) error {
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
