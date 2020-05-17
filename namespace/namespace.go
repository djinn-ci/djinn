// Package namespace provides model implementations for the Namespace entity
// and its related entities.
// 
// Each entity implemented, has a corresponding Store which is used for
// creating, updating, and deleting models, as well as querying them from the
// database. These methods wrap the onces provided by the model package via
// model.Store.
//
// Entities:
// - Collaborator
// - Invite
// - Namespace
package namespace

import (
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type Namespace struct {
	ID          int64
	UserID      int64         `db:"user_id"`
	RootID      sql.NullInt64 `db:"root_id"`
	ParentID    sql.NullInt64 `db:"parent_id"`
	Name        string        `db:"name"`
	Path        string        `db:"path"`
	Description string        `db:"description"`
	Level       int64         `db:"level"`
	Visibility  Visibility    `db:"visibility"`
	CreatedAt   time.Time     `db:"created_at"`

	User          *user.User         `db:"-"`
	Parent        *Namespace         `db:"-"`
	Build         model.Model        `db:"-"`
	Collaborators map[int64]struct{} `db:"-"`
}

// Resource represents a model that can exist within a Namespace, such as an
// object, image, variable, or key.
type Resource struct {
	// User is the authenticated User modifying/creating the resource in the
	// Namespace.
	User *user.User `schema:"-"`

	Namespaces *Store `schema:"-"`
	Namespace  string `schema:"namespace"`
}

type Store struct {
	model.Store

	User      *user.User
	Namespace *Namespace
}

var (
	_ model.Model  = (*Namespace)(nil)
	_ model.Loader = (*Store)(nil)
	_ model.Binder = (*Store)(nil)

	table  = "namespaces"
	rename = regexp.MustCompile("^[a-zA-Z0-9]+$")

	MaxDepth int64 = 20
	ErrDepth       = errors.New("namespace cannot exceed depth of 20")
	ErrName        = errors.New("namespace name can only contain letters and numbers")
	ErrPermission  = errors.New("permission denied")
)

// NewStore returns a new Store for querying the namespaces table. Each model
// passed to this function will be bound to the returned Store.
func NewStore(db *sqlx.DB, mm ...model.Model) *Store {
	s := &Store{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// Model is called along with model.Slice to convert the given slice of
// Namespace models to a slice of model.Model interfaces.
func Model(nn []*Namespace) func(int) model.Model {
	return func(i int) model.Model {
		return nn[i]
	}
}

// SelectRootId returns query.Query that will return the root_id of the
// Namespace by the given id.
func SelectRootID(id int64) query.Query {
	return query.Select(
		query.Columns("root_id"),
		query.From(table),
		query.Where("id", "=", id),
	)
}

// SharedWith expects the given model.Model to be of *user.User. This will
// apply two WHERE clauses to the query, that will check if the given User
// model either owns the Namespace, or is a collaborator for the Namespace. If
// the given model.Model is not of *user.User, then the WHERE clauses are not
// applied.
func SharedWith(m model.Model) query.Option {
	return func(q query.Query) query.Query {
		if _, ok := m.(*user.User); !ok {
			return q
		}
		if m == nil || m.IsZero() {
			return q
		}
		_, id := m.Primary()
		return query.Options(
			query.Where("user_id", "=", id),
			query.OrWhereQuery("root_id", "IN",
				query.Select(
					query.Columns("namespace_id"),
					query.From(collaboratorTable),
					query.Where("user_id", "=", id),
				),
			),
		)(q)
	}
}

// BindNamespace will get the Namespace by the path specified in the Namespace
// field of the Resource struct. It will then check to see if the Resource User
// specified via the User field can access the returned Namespace. If the User
// has access, then the Namespace is bound to the given model.Binder. If the
// User does not have access to the Namespace then ErrPermission is returned.
// If the Namespace field is empty, then this method does nothing, and returns
// nil.
func (r Resource) BindNamespace(b model.Binder) error {
	if r.Namespace == "" {
		return nil
	}

	n, err := r.Namespaces.GetByPath(r.Namespace)

	if err != nil {
		return errors.Err(err)
	}

	if n.Collaborators != nil && !n.CanAdd(r.User) {
		return ErrPermission
	}

	b.Bind(n)
	return nil
}

// AccessibleBy assumes that the given model.Model is of type *user.User, and
// checks that the given model.Model has access to the current Namespace,
// depending on the Namespace's visibility. If the Namespace visibility is
// Public, then this method will return true. If the Namespace visibility is
// Private, then it will return true if the given model.Model is not zero. If
// the visibility is Private, then it will check to see if the primary key of
// the model.Model exists in the Namespace collaborators (this assumes the
// collaborators have been previously loaded into the Namespace). If the given
// model.Model is not of type *user.User then this method returns false.
func (n *Namespace) AccessibleBy(m model.Model) bool {
	if _, ok := m.(*user.User); !ok {
		return false
	}

	_, id := m.Primary()

	switch n.Visibility {
	case Public:
		return true
	case Internal:
		return !m.IsZero()
	case Private:
		_, ok := n.Collaborators[id]

		if !ok {
			ok = n.UserID == id
		}
		return ok
	default:
		return false
	}
}

// CanAdd assumes that the given model.Model is of type *user.User, and checks
// to see if that User either owns the Namespace, or is a Collaborator in that
// Namespace. This assumes that the collaborators have been previously loaded
// into the Namespace.
func (n *Namespace) CanAdd(m model.Model) bool {
	if _, ok := m.(*user.User); !ok {
		return false
	}

	_, id := m.Primary()
	_, ok := n.Collaborators[id]

	if !ok {
		ok = n.UserID == id
	}
	return ok
}

// CascadeVisibility applies the visibility of the root namespace to all child
// namespaces. This will have no effect if called on a child namespace.
func (n *Namespace) CascadeVisibility(db *sqlx.DB) error {
	if n.ID != n.RootID.Int64 {
		return nil
	}

	q := query.Update(
		query.Table(table),
		query.Set("visibility", n.Visibility),
		query.Where("root_id", "=", n.RootID),
	)

	stmt, err := db.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(q.Args()...)
	return errors.Err(err)
}

func (n *Namespace) IsZero() bool {
	return n == nil || n.ID == 0 &&
		n.UserID == 0 &&
		!n.ParentID.Valid &&
		n.Name == "" &&
		n.Path == ""&&
		n.Description == "" &&
		n.Level == 0 &&
		n.Visibility == Visibility(0) &&
		n.CreatedAt == time.Time{}
}

// LoadCollaborators takes the given slice of Collaborator models and adds them
// to the Collaborators map on the current Namespace, if the Namespace ID for
// the Collaborator matches the current Namespace ID.
func (n *Namespace) LoadCollaborators(cc []*Collaborator) {
	if n.Collaborators == nil {
		n.Collaborators = make(map[int64]struct{})
	}

	for _, c := range cc {
		if c.NamespaceID != n.ID {
			continue
		}
		n.Collaborators[c.UserID] = struct{}{}
	}
}

// Bind the given models to the current Namespace. This will only bind the
// model if they are one of the following,
//
// - *user.User
// - *namespace.Namespace
//
// The bound Namespace will be treated as the parent of the current Namespace,
// and set on the Parent field.
func (n *Namespace) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			n.User = m.(*user.User)
		case *Namespace:
			n.Parent = m.(*Namespace)
		}
	}
}

func (n *Namespace) SetPrimary(id int64) {
	n.ID = id
}

func (n *Namespace) Primary() (string, int64) { return "id", n.ID }

// Endpoint returns the endpoint for the current Namespace, appending the given
// variadic URIs. The Namespace endpoint will contain the username of the bound
// User model. If there is no bound User model, or the model IsZero, then an
// empty string is returned.
func (n *Namespace) Endpoint(uri ...string) string {
	if n.User == nil || n.User.IsZero() {
		return ""
	}

	username, _ := n.User.Values()["username"]
	endpoint := fmt.Sprintf("/n/%s/%s", username, n.Path)

	if len(uri) > 0 {
		return fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}
	return endpoint
}

func (n *Namespace) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":     n.UserID,
		"root_id":     n.RootID,
		"parent_id":   n.ParentID,
		"name":        n.Name,
		"path":        n.Path,
		"description": n.Description,
		"level":       n.Level,
		"visibility":  n.Visibility,
	}
}

// Bind the given models to the current Namespace. This will only bind the
// model if they are one of the following,
//
// - *user.User
// - *namespace.Namespace
//
// The bound Namespace model will be treated as the parent Namespace.
func (s *Store) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		case *Namespace:
			s.Namespace = m.(*Namespace)
		}
	}
}

// Create inserts the given Namespace models into the namespaces table.
func (s *Store) Create(nn ...*Namespace) error {
	models := model.Slice(len(nn), Model(nn))
	return errors.Err(s.Store.Create(table, models...))
}

// Update updates the given Namespace models in the namespaces table.
func (s *Store) Update(nn ...*Namespace) error {
	models := model.Slice(len(nn), Model(nn))
	return errors.Err(s.Store.Update(table, models...))
}

// Delete deletes the given Namespace models, and their children from the
// namespaces table.
func (s *Store) Delete(nn ...*Namespace) error {
	models := model.Slice(len(nn), Model(nn))

	ids := model.MapKey("id", models)

	q := query.Delete(query.From(table), query.Where("root_id", "IN", ids...))

	stmt, err := s.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(q.Args()...)
	return errors.Err(err)
}


// Paginate returns the model.Paginator for the namespaces table for the given
// page. This applies the SharedWith option on the bound User model, and the
// model.Where option to the bound Namespace model on the "parent_id".
func (s Store) Paginate(page int64, opts ...query.Option) (model.Paginator, error) {
	paginator, err := s.Store.Paginate(table, page, opts...)
	return paginator, errors.Err(err)
}

// New returns a new Namespace binding any non-nil models to it from the current
// Store.
func (s *Store) New() *Namespace {
	n := &Namespace{
		User:   s.User,
		Parent: s.Namespace,
	}

	if s.User != nil {
		n.UserID = s.User.ID
	}

	if s.Namespace != nil {
		n.ParentID = sql.NullInt64{
			Int64: s.Namespace.ID,
			Valid: true,
		}
	}
	return n
}

// All returns a slice of Namespace models, applying each query.Option that is
// given. The SharedWith option is applied to the bound User model, and the
// model.Where option to the bound Namespace model on the "parent_id".
func (s *Store) All(opts ...query.Option) ([]*Namespace, error) {
	nn := make([]*Namespace, 0)

	opts = append([]query.Option{
		SharedWith(s.User),
		model.Where(s.Namespace, "parent_id"),
	}, opts...)

	err := s.Store.All(&nn, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, n := range nn {
		n.User = s.User
	}
	return nn, errors.Err(err)
}

// Index returns the paginated results from the namespaces table depending on
// the values that are present in url.Values. Detailed below are the values
// that are used from the given url.Values,
//
// path - This applies the model.Search query.Option using the value os path
func (s *Store) Index(vals url.Values, opts ...query.Option) ([]*Namespace, model.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		model.Search("path", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, opts...)

	if err != nil {
		return []*Namespace{}, paginator, errors.Err(err)
	}

	nn, err := s.All(append(
		opts,
		query.OrderAsc("path"),
		query.Limit(model.PageLimit),
		query.Offset(paginator.Offset),
	)...)
	return nn, paginator, errors.Err(err)
}

// Get returns a single Namespace model, applying each query.Option that is
// given. The SharedWith option is applied to the bound User model, and the
// model.Where option to the bound Namespace model on the "parent_id".
func (s *Store) Get(opts ...query.Option) (*Namespace, error) {
	n := &Namespace{
		User:   s.User,
		Parent: s.Namespace,
	}

	opts = append([]query.Option{
		SharedWith(s.User),
		model.Where(s.Namespace, "parent_id"),
	}, opts...)

	err := s.Store.Get(n, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return n, errors.Err(err)
}

// GetByPath returns a Namespace by the given path. If it does not exist then
// it will be created, including all grandparents up to the maximum possible
// depth for a Namespace. If any invalid names for a Namespace appear in the
// path, then the child Namespaces will not be created beyong that point. If
// a '@' is present in the given path, then the Namespace for that given user
// will be returned if it exists, if not then nothing will be returned and a
// Namespace will not be created.
func (s *Store) GetByPath(path string) (*Namespace, error) {
	create := true
	users := user.NewStore(s.DB)

	if strings.Contains(path, "@") {
		parts := strings.Split(path, "@")

		username := parts[1]
		path = parts[0]

		u, err := users.Get(query.Where("username", "=", username))

		if err != nil {
			return &Namespace{}, errors.Err(err)
		}

		if !u.IsZero() {
			s.User = u
			create = false
		}
	}

	n, err := s.Get(query.Where("path", "=", path))

	if err != nil {
		return n, errors.Err(err)
	}

	if !n.IsZero() {
		cc, err := NewCollaboratorStore(s.DB, n).All()

		if err != nil {
			return n, errors.Err(err)
		}
		n.LoadCollaborators(cc)
		return n, nil
	}

	if !create {
		return n, nil
	}

	p := &Namespace{}

	parts := strings.Split(path, "/")

	for i, name := range parts {
		if p.Level + 1 > MaxDepth {
			break
		}

		if !rename.Match([]byte(name)) {
			if i == 0 && len(parts) == 1 {
				return n, ErrName
			}
			break
		}

		n = s.New()
		n.Name = name
		n.Path = name
		n.Level = p.Level + 1
		n.Visibility = Private

		if !p.IsZero() {
			n.RootID = p.RootID
			n.ParentID = sql.NullInt64{
				Int64: p.ID,
				Valid: true,
			}
			n.Path = strings.Join([]string{p.Path, n.Name}, "/")
		}

		if err := s.Create(n); err != nil {
			return n, errors.Err(err)
		}

		if p.IsZero() {
			n.RootID = sql.NullInt64{
				Int64: n.ID,
				Valid: true,
			}

			if err := s.Update(n); err != nil {
				return n, errors.Err(err)
			}
		}

		p = n
	}
	return n, nil
}

// Load loads in a slice of Namespace models where the given key is in the list
// of given vals. Each model is loaded individually via a call to the given
// load callback. This method calls Store.All under the hood, so any
// bound models will impact the models being loaded.
func (s *Store) Load(key string, vals []interface{}, load model.LoaderFunc) error {
	nn, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, n := range nn {
			load(i, n)
		}
	}
	return nil
}
