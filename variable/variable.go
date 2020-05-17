package variable

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
)

type Variable struct {
	ID          int64         `db:"id"`
	UserID      int64         `db:"user_id"`
	NamespaceID sql.NullInt64 `db:"namespace_id"`
	Key         string        `db:"key"`
	Value       string        `db:"value"`
	CreatedAt   time.Time     `db:"created_at"`

	User      *user.User           `db:"-"`
	Namespace *namespace.Namespace `db:"-"`
}

type Store struct {
	model.Store

	User      *user.User
	Namespace *namespace.Namespace
}

var (
	_ model.Model  = (*Variable)(nil)
	_ model.Binder = (*Store)(nil)
	_ model.Loader = (*Store)(nil)

	table     = "variables"
	relations = map[string]model.RelationFunc{
		"namespace": model.Relation("namespace_id", "id"),
	}
)

// NewStore returns a new Store for querying the variables table. Each of the
// given models is bound to the returned Store.
func NewStore(db *sqlx.DB, mm ...model.Model) *Store {
	s := &Store{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// Model is called along with model.Slice to convert the given slice of
// Variable  models to a slice of model.Model interfaces.
func Model(vv []*Variable) func(int) model.Model {
	return func(i int) model.Model {
		return vv[i]
	}
}

// LoadRelations loads all of the available relations for the given Variable 
// models using the given loaders available.
func LoadRelations(loaders model.Loaders, vv ...*Variable) error {
	mm := model.Slice(len(vv), Model(vv))
	return errors.Err(model.LoadRelations(relations, loaders, mm...))
}

// Bind the given models to the current Variable. This will only bind the model
// if they are one of the following,
//
// - *user.User
// - *namespace.Namespace
func (v *Variable) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			v.User = m.(*user.User)
		case *namespace.Namespace:
			v.Namespace = m.(*namespace.Namespace)
		}
	}
}

func (v *Variable) SetPrimary(i int64) {
	v.ID = i
}

func (v *Variable) Primary() (string, int64) {
	return "id", v.ID
}

// Endpoint returns the endpoint to the current Variable model, with the given
// URI parts appended to it.
func (v *Variable) Endpoint(uri ...string) string {
	endpoint := fmt.Sprintf("/variables/%v", v.ID)

	if len(uri) > 0 {
		return fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}
	return endpoint
}

func (v *Variable) IsZero() bool {
	return v == nil || v.ID == 0 &&
		v.UserID == 0 &&
		!v.NamespaceID.Valid &&
		v.Key == "" &&
		v.Value == "" &&
		v.CreatedAt == time.Time{}
}

func (v *Variable) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":      v.UserID,
		"namespace_id": v.NamespaceID,
		"key":          v.Key,
		"value":        v.Value,
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

// Create inserts the given Variable models into the variables table.
func (s *Store) Create(vv ...*Variable) error {
	models := model.Slice(len(vv), Model(vv))
	return s.Store.Create(table, models...)
}

// Delete delets the given Variable models from the variables table.
func (s *Store) Delete(vv ...*Variable) error {
	models := model.Slice(len(vv), Model(vv))
	return s.Store.Delete(table, models...)
}

// Paginate returns the model.Paginator for the variables table for the given
// page. This applies the namespace.WhereCollaborator option to the *user.User
// bound model, and the model.Where option to the *namespace.Namespace bound
// model.
func (s *Store) Paginate(page int64, opts ...query.Option) (model.Paginator, error) {
	paginator, err := s.Store.Paginate(table, page, opts...)
	return paginator, errors.Err(err)
}

// New returns a new Variable binding any non-nil models to it from the current
// Store.
func (s *Store) New() *Variable {
	v := &Variable{
		User:      s.User,
		Namespace: s.Namespace,
	}

	if s.User != nil {
		v.UserID = s.User.ID
	}

	if s.Namespace != nil {
		v.NamespaceID = sql.NullInt64{
			Int64: s.Namespace.ID,
			Valid: true,
		}
	}
	return v
}

// All returns a slice of Variable models, applying each query.Option that is
// given. The namespace.WhereCollaborator option is applied to the *user.User
// bound model, and the model.Where option is applied to the
// *namespace.Namespace bound model.
func (s *Store) All(opts ...query.Option) ([]*Variable, error) {
	vv := make([]*Variable, 0)

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.All(&vv, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, v := range vv {
		v.User = s.User
		v.Namespace = s.Namespace
	}
	return vv, errors.Err(err)
}

// Index returns the paginated results from the variables table depending on the
// values that are present in url.Values. Detailed below are the values that
// are used from the given url.Values,
//
// key - This applies the model.Search query.Option using the value of key
func (s *Store) Index(vals url.Values, opts ...query.Option) ([]*Variable, model.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		model.Search("key", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, opts...)

	if err != nil {
		return []*Variable{}, paginator, errors.Err(err)
	}

	vv, err := s.All(append(
		opts,
		query.OrderAsc("key"),
		query.Limit(model.PageLimit),
		query.Offset(paginator.Offset),
	)...)
	return vv, paginator, errors.Err(err)
}

// All returns a single Variable model, applying each query.Option that is
// given. The namespace.WhereCollaborator option is applied to the *user.User
// bound model, and the model.Where option is applied to the
// *namespace.Namespace bound model.
func (s *Store) Get(opts ...query.Option) (*Variable, error) {
	v := &Variable{
		User:      s.User,
		Namespace: s.Namespace,
	}

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.Get(v, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return v, errors.Err(err)
}

// Load loads in a slice of Variable models where the given key is in the list
// of given vals. Each model is loaded individually via a call to the given
// load callback. This method calls Store.All under the hood, so any
// bound models will impact the models being loaded.
func (s *Store) Load(key string, vals []interface{}, load model.LoaderFunc) error {
	vv, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, v := range vv {
			load(i, v)
		}
	}
	return nil
}
