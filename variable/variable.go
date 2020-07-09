// Package object implements the database.Model interface for the Variable
// entity.
package variable

import (
	"context"
	"database/sql"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Variable is the type that represents a variable that has been set by a user.
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

// Store is the type for creating and modifying Variable models in the database.
type Store struct {
	database.Store

	// User is the bound user.User model. If not nil this will bind the
	// user.User model to any Variable models that are created. If not nil this
	// will append a WHERE clause on the user_id column for all SELECT queries
	// performed.
	User *user.User

	// Namespace is the bound namespace.Namespace model. If not nil this will
	// bind the namespace.Namespace model to any Variable models that are
	// created. If not nil this will append a WHERE clause on the namespace_id
	// column for all SELECT queries performed.
	Namespace *namespace.Namespace
}

var (
	_ database.Model  = (*Variable)(nil)
	_ database.Binder = (*Store)(nil)
	_ database.Loader = (*Store)(nil)

	table     = "variables"
	relations = map[string]database.RelationFunc{
		"namespace": database.Relation("namespace_id", "id"),
	}
)

// NewStore returns a new Store for querying the variables table. Each of the
// given models is bound to the returned Store.
func NewStore(db *sqlx.DB, mm ...database.Model) *Store {
	s := &Store{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// FromContext returns the Variable model from the given context, if any.
func FromContext(ctx context.Context) (*Variable, bool) {
	v, ok := ctx.Value("variable").(*Variable)
	return v, ok
}

// Model is called along with database.ModelSlice to convert the given slice of
// Variable  models to a slice of database.Model interfaces.
func Model(vv []*Variable) func(int) database.Model {
	return func(i int) database.Model {
		return vv[i]
	}
}

// LoadRelations loads all of the available relations for the given Variable 
// models using the given loaders available.
func LoadRelations(loaders *database.Loaders, vv ...*Variable) error {
	mm := database.ModelSlice(len(vv), Model(vv))
	return errors.Err(database.LoadRelations(relations, loaders, mm...))
}

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either user.User or namespace.Namespace.
func (v *Variable) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			v.User = m.(*user.User)
		case *namespace.Namespace:
			v.Namespace = m.(*namespace.Namespace)
		}
	}
}

// SetPrimary implements the database.Model interface.
func (v *Variable) SetPrimary(i int64) { v.ID = i }

// Primary implements the database.Model interface.
func (v *Variable) Primary() (string, int64) { return "id", v.ID }

// Endpoint returns the endpoint to the current Variable database, with the given
// URI parts appended to it.
func (v *Variable) Endpoint(uri ...string) string {
	if len(uri) > 0 {
		return "/variables/" + strconv.FormatInt(v.ID, 10) + "/" + strings.Join(uri, "/")
	}
	return "/variables/" + strconv.FormatInt(v.ID, 10)
}

// IsZero implements the database.Model interface.
func (v *Variable) IsZero() bool {
	return v == nil || v.ID == 0 &&
		v.UserID == 0 &&
		!v.NamespaceID.Valid &&
		v.Key == "" &&
		v.Value == "" &&
		v.CreatedAt == time.Time{}
}

// JSON implements the database.Model interface. This will return a map with
// the current Variable values under each key. If any of the User, or Namespace
// bound models exist on the Variable, then the JSON representation of these
// models will be returned in the map, under the user, and namespace keys
// respectively.
func (v *Variable) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":           v.ID,
		"user_id":      v.UserID,
		"namespace_id": nil,
		"key":          v.Key,
		"value":        v.Value,
		"created_at":   v.CreatedAt.Format(time.RFC3339),
		"url":          addr + v.Endpoint(),
	}

	if v.NamespaceID.Valid {
		json["namespace_id"] = v.NamespaceID.Int64
	}

	for name, m := range map[string]database.Model{
		"user":      v.User,
		"namespace": v.Namespace,
	}{
		if !m.IsZero() {
			json[name] = m.JSON(addr)
		}
	}
	return json
}

// Values implements the database.Model interface. This will return a map with
// the following values, user_id, namespace_id, key, and value.
func (v *Variable) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":      v.UserID,
		"namespace_id": v.NamespaceID,
		"key":          v.Key,
		"value":        v.Value,
	}
}

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either user.User or namespace.Namespace.
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

// Create creates a new Variable model with the given key and value.
func (s *Store) Create(key, val string) (*Variable, error) {
	v := s.New()
	v.Key = key
	v.Value = val

	err := s.Store.Create(table, v)
	return v, errors.Err(err)
}

// Delete deletes the Variable models from the database with the given ids.
func (s *Store) Delete(ids ...int64) error {
	mm := make([]database.Model, 0, len(ids))

	for _, id := range ids {
		mm = append(mm, &Variable{ID: id})
	}
	return errors.Err(s.Store.Delete(table, mm...))
}

// Paginate returns the database.Paginator for the variables table for the given
// page. This applies the namespace.WhereCollaborator option to the *user.User
// bound database, and the database.Where option to the *namespace.Namespace bound
// database.
func (s *Store) Paginate(page int64, opts ...query.Option) (database.Paginator, error) {
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
// bound database, and the database.Where option is applied to the
// *namespace.Namespace bound database.
func (s *Store) All(opts ...query.Option) ([]*Variable, error) {
	vv := make([]*Variable, 0)

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		database.Where(s.Namespace, "namespace_id"),
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
// key - This applies the database.Search query.Option using the value of key
func (s *Store) Index(vals url.Values, opts ...query.Option) ([]*Variable, database.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		database.Search("key", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, opts...)

	if err != nil {
		return []*Variable{}, paginator, errors.Err(err)
	}

	vv, err := s.All(append(
		opts,
		query.OrderAsc("key"),
		query.Limit(database.PageLimit),
		query.Offset(paginator.Offset),
	)...)
	return vv, paginator, errors.Err(err)
}

// All returns a single Variable database, applying each query.Option that is
// given. The namespace.WhereCollaborator option is applied to the *user.User
// bound database, and the database.Where option is applied to the
// *namespace.Namespace bound database.
func (s *Store) Get(opts ...query.Option) (*Variable, error) {
	v := &Variable{
		User:      s.User,
		Namespace: s.Namespace,
	}

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		database.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.Get(v, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return v, errors.Err(err)
}

// Load loads in a slice of Variable models where the given key is in the list
// of given vals. Each database is loaded individually via a call to the given
// load callback. This method calls Store.All under the hood, so any
// bound models will impact the models being loaded.
func (s *Store) Load(key string, vals []interface{}, load database.LoaderFunc) error {
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
