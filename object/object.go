// Package object implements the model.Model interface for the Object entity.
package object

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type Object struct {
	ID          int64         `db:"id"`
	UserID      int64         `db:"user_id"`
	NamespaceID sql.NullInt64 `db:"namespace_id"`
	Hash        string        `db:"hash"`
	Name        string        `db:"name"`
	Type        string        `db:"type"`
	Size        int64         `db:"size"`
	MD5         []byte        `db:"md5"`
	SHA256      []byte        `db:"sha256"`
	CreatedAt   time.Time     `db:"created_at"`
	DeletedAt   pq.NullTime   `db:"deleted_at"`

	User      *user.User           `db:"-"`
	Namespace *namespace.Namespace `db:"-"`
}

type Store struct {
	model.Store

	User      *user.User
	Namespace *namespace.Namespace
}

var (
	_ model.Model  = (*Object)(nil)
	_ model.Binder = (*Store)(nil)
	_ model.Loader = (*Store)(nil)

	table     = "objects"
	relations = map[string]model.RelationFunc{
		"namespace": model.Relation("namespace_id", "id"),
	}
)

// NewStore returns a new Store for querying the objects table. Each model
// passed to this function will be bound to the returned Store.
func NewStore(db *sqlx.DB, mm ...model.Model) *Store {
	s := &Store{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// LoadRelations loads all of the available relations for the given Object
// models using the given loaders available.
func LoadRelations(loaders model.Loaders, oo ...*Object) error {
	mm := model.Slice(len(oo), Model(oo))
	return errors.Err(model.LoadRelations(relations, loaders, mm...))
}

// Model is called along with model.Slice to convert the given slice of Object
// models to a slice of model.Model interfaces.
func Model(oo []*Object) func(int)model.Model {
	return func(i int) model.Model {
		return oo[i]
	}
}

// Bind the given models to the current Object. This will only bind the model if
// they are one of the following,
//
// - *user.User
// - *namespace.Namespace
func (o *Object) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			o.User = m.(*user.User)
		case *namespace.Namespace:
			o.Namespace = m.(*namespace.Namespace)
		}
	}
}

func (o *Object) SetPrimary(id int64) {
	o.ID = id
}

func (o *Object) Primary() (string, int64) { return "id", o.ID }

// Endpoint returns the endpoint for the current Object. Each URI part in the
// given variadic list will be appended to the final returned string.
func (o *Object) Endpoint(uri ...string) string {
	endpoint := fmt.Sprintf("/objects/%v", o.ID)

	if len(uri) > 0 {
		return fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}
	return endpoint
}

func (o *Object) IsZero() bool {
	return o == nil || o.ID == 0 &&
		o.UserID == 0 &&
		!o.NamespaceID.Valid &&
		o.Hash == "" &&
		o.Name == "" &&
		o.Type == "" &&
		o.Size == 0 &&
		len(o.MD5) == 0 &&
		len(o.SHA256) == 0 &&
		o.CreatedAt == time.Time{} &&
		!o.DeletedAt.Valid
}

func (o *Object) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":      o.UserID,
		"namespace_id": o.NamespaceID,
		"hash":         o.Hash,
		"name":         o.Name,
		"type":         o.Type,
		"size":         o.Size,
		"md5":          o.MD5,
		"sha256":       o.SHA256,
		"deleted_at":   o.DeletedAt,
	}
}

// Bind the given models to the current Store. This will only bind the model if
// they are one of the following,
//
// - *user.User
// - *namespace.Namespace
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

// Create inserts the given Object models into the objects table.
func (s *Store) Create(oo ...*Object) error {
	models := model.Slice(len(oo), Model(oo))
	return errors.Err(s.Store.Create(table, models...))
}

// Update updates the given Object models in the objects table.
func (s *Store) Update(oo ...*Object) error {
	models := model.Slice(len(oo), Model(oo))
	return errors.Err(s.Store.Update(table, models...))
}

// Delete deletes the given Object models from the objects table.
func (s *Store) Delete(oo ...*Object) error {
	models := model.Slice(len(oo), Model(oo))
	return errors.Err(s.Store.Delete(table, models...))
}

// Paginate returns the model.Paginator for the objects table for the given
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

// New returns a new Object binding any non-nil models to it from the current
// Store.
func (s *Store) New() *Object {
	o := &Object{
		User:      s.User,
		Namespace: s.Namespace,
	}

	if s.User != nil {
		o.UserID = s.User.ID
	}

	if s.Namespace != nil {
		o.NamespaceID = sql.NullInt64{
			Int64: s.Namespace.ID,
			Valid: true,
		}
	}
	return o
}

// All returns a slice of Object models, applying each query.Option that is
// given. The namespace.WhereCollaborator option is applied to the *user.User
// bound model, and the model.Where option is applied to the
// *namespace.Namespace bound model.
func (s *Store) All(opts ...query.Option) ([]*Object, error) {
	oo := make([]*Object, 0)

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.All(&oo, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, o := range oo {
		o.User = s.User
		o.Namespace = s.Namespace
	}
	return oo, errors.Err(err)
}

// Index returns the paginated results from the objects table depending on the
// values that are present in url.Values. Detailed below are the values that
// are used from the given url.Values,
//
// name - This applies the model.Search query.Option using the value of name 
func (s *Store) Index(vals url.Values, opts ...query.Option) ([]*Object, model.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		model.Search("name", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, opts...)

	if err != nil {
		return []*Object{}, paginator, errors.Err(err)
	}

	oo, err := s.All(append(
		opts,
		query.OrderAsc("name"),
		query.Limit(model.PageLimit),
		query.Offset(paginator.Offset),
	)...)
	return oo, paginator, errors.Err(err)
}

// Load loads in a slice of Object models where the given key is in the list
// of given vals. Each model is loaded individually via a call to the given
// load callback. This method calls Store.All under the hood, so any
// bound models will impact the models being loaded.
func (s *Store) Load(key string, vals []interface{}, load model.LoaderFunc) error {
	oo, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, o := range oo {
			load(i, o)
		}
	}
	return nil
}

// Get returns a single Object model, applying each query.Option that is given.
// The namespace.WhereCollaborator option is applied to the *user.User bound
// model, and the model.Where option is applied to the *namespace.Namespace
// bound model.
func (s *Store) Get(opts ...query.Option) (*Object, error) {
	o := &Object{
		User:      s.User,
		Namespace: s.Namespace,
	}

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.Get(o, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return o, errors.Err(err)
}
