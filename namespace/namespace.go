// Package namespace provides database implementations for the Namespace entity
// and its related entities.
package namespace

import (
	"context"
	"database/sql"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Namespace is the type that represents a namespace in the system for grouping
// related resources together.
type Namespace struct {
	ID          int64         `db:"id"`
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
	Build         database.Model     `db:"-"`
	Collaborators map[int64]struct{} `db:"-"`
}

// Resource is the type that represents a model that can exist within a
// Namespace, such as an object, image, variable, or key.
type Resource struct {
	Owner  *user.User `schema:"-"` // Owner is who the resource belongs to once authored
	Author *user.User `schema:"-"` // Author is the original author of the resource

	Namespaces *Store `schema:"-"`
	Namespace  string `schema:"namespace"`
}

// Store is the type for creating and modifying Namespace models in the
// database.
type Store struct {
	database.Store

	// User is the bound User model. If not nil this will bind the User model to
	// any Namespace models that are created.
	User *user.User

	// Namespace is the bound Namespace model. If not nil this will bind the
	// Namespace model to any Build models that are created. If not nil this
	// will append a WHERE clause on the parent_id column for all SELECT
	// queries performed.
	Namespace *Namespace
}

var (
	_ database.Model  = (*Namespace)(nil)
	_ database.Loader = (*Store)(nil)
	_ database.Binder = (*Store)(nil)

	table  = "namespaces"
	rename = regexp.MustCompile("^[a-zA-Z0-9]+$")

	MaxDepth      int64 = 20
	ErrDepth            = errors.New("namespace cannot exceed depth of 20")
	ErrName             = errors.New("namespace name can only contain letters and numbers")
	ErrPermission       = errors.New("namespace permissions invalid")
	ErrDeleteSelf       = errors.New("cannot delete self from namespace")
)

// NewStore returns a new Store for querying the namespaces table. Each database
// passed to this function will be bound to the returned Store.
func NewStore(db *sqlx.DB, mm ...database.Model) *Store {
	s := &Store{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// FromContext returns the Namespace model from the given context, if any.
func FromContext(ctx context.Context) (*Namespace, bool) {
	n, ok := ctx.Value("namespace").(*Namespace)
	return n, ok
}

// Model is called along with database.ModelSlice to convert the given slice of
// Namespace models to a slice of database.Model interfaces.
func Model(nn []*Namespace) func(int) database.Model {
	return func(i int) database.Model {
		return nn[i]
	}
}

// SelectRootId returns query.Query that will return the root_id of the
// Namespace by the given id.
func SelectRootID(id int64) query.Query {
	return query.Select(
		query.Columns("root_id"),
		query.From(table),
		query.Where("id", "=", query.Arg(id)),
	)
}

// SharedWith expects the given database.Model to be of *user.User. This will
// apply two WHERE clauses to the query, that will check if the given User
// database either owns the Namespace, or is a collaborator for the Namespace. If
// the given database.Model is not of *user.User, then the WHERE clauses are not
// applied.
func SharedWith(u *user.User) query.Option {
	return func(q query.Query) query.Query {
		if u == nil {
			return q
		}

		return query.Options(
			query.Where("user_id", "=", query.Arg(u.ID)),
			query.OrWhere("root_id", "IN",
				query.Select(
					query.Columns("namespace_id"),
					query.From(collaboratorTable),
					query.Where("user_id", "=", query.Arg(u.ID)),
				),
			),
		)(q)
	}
}

// Resolve will get the namespace and namespace owner for the current resource.
// This will bind the namespace and namespace owner to the given
// database.Binder. If the resource author cannot add the resource to the
// namespace, then this return ErrPermission.
func (r *Resource) Resolve(b database.Binder) error {
	if r.Namespace == "" {
		return nil
	}

	n, err := r.Namespaces.GetByPath(r.Namespace)

	if err != nil {
		return errors.Err(err)
	}

	if n.Collaborators != nil {
		if !n.CanAdd(r.Author) {
			return ErrPermission
		}
	}

	b.Bind(n, n.User)
	return nil
}

// AccessibleBy assumes that the given database.Model is of type *user.User, and
// checks that the given database.Model has access to the current Namespace,
// depending on the Namespace's visibility. If the Namespace visibility is
// Public, then this method will return true. If the Namespace visibility is
// Private, then it will return true if the given database.Model is not zero. If
// the visibility is Private, then it will check to see if the primary key of
// the database.Model exists in the Namespace collaborators (this assumes the
// collaborators have been previously loaded into the Namespace). If the given
// database.Model is not of type *user.User then this method returns false.
func (n *Namespace) AccessibleBy(m database.Model) bool {
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

// CanAdd assumes that the given database.Model is of type *user.User, and checks
// to see if that User either owns the Namespace, or is a Collaborator in that
// Namespace. This assumes that the collaborators have been previously loaded
// into the Namespace.
func (n *Namespace) CanAdd(m database.Model) bool {
	if m.IsZero() {
		return false
	}

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

// IsZero implements the database.Model interface.
func (n *Namespace) IsZero() bool {
	return n == nil || n.ID == 0 &&
		n.UserID == 0 &&
		!n.ParentID.Valid &&
		n.Name == "" &&
		n.Path == "" &&
		n.Description == "" &&
		n.Level == 0 &&
		n.Visibility == Visibility(0) &&
		n.CreatedAt == time.Time{}
}

// JSON implements the database.Model interface. This will return a map with
// the current Namespace values under each key. This will also include urls to
// the Namespace's builds, namespaces, images, objects, variables, keys, and
// collaborators. I any of the User, Parent, or Build models exist on the
// Namespace then the JSON representation of these models will be in the
// returned map, under the user, parent, and build keys respectively.
func (n *Namespace) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":                n.ID,
		"user_id":           n.UserID,
		"root_id":           n.RootID.Int64,
		"parent_id":         nil,
		"name":              n.Name,
		"path":              n.Path,
		"description":       n.Description,
		"visibility":        n.Visibility.String(),
		"created_at":        n.CreatedAt.Format(time.RFC3339),
		"url":               addr + n.Endpoint(),
		"builds_url":        addr + n.Endpoint("builds"),
		"namespaces_url":    addr + n.Endpoint("namespaces"),
		"images_url":        addr + n.Endpoint("images"),
		"objects_url":       addr + n.Endpoint("objects"),
		"variables_url":     addr + n.Endpoint("variables"),
		"keys_url":          addr + n.Endpoint("keys"),
		"collaborators_url": addr + n.Endpoint("collaborators"),
		"webhooks_url":      addr + n.Endpoint("webhooks"),
	}

	if n.ParentID.Valid {
		json["parent_id"] = n.ParentID.Int64
	}

	for name, m := range map[string]database.Model{
		"user":   n.User,
		"parent": n.Parent,
		"build":  n.Build,
	} {
		if m != nil && !m.IsZero() {
			json[name] = m.JSON(addr)
		}
	}
	return json
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

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either user.User, or namespace.Namespace models.
func (n *Namespace) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *user.User:
			n.User = v
		case *Namespace:
			n.Parent = v
		}
	}
}

// SetPrimary implements the database.Model interface.
func (n *Namespace) SetPrimary(id int64) { n.ID = id }

// Primary implements the database.Model interface.
func (n *Namespace) Primary() (string, int64) { return "id", n.ID }

// Endpoint returns the endpoint for the current Namespace, appending the given
// variadic URIs. The Namespace endpoint will contain the username of the bound
// User database. If there is no bound User database, or the database IsZero, then an
// empty string is returned.
func (n *Namespace) Endpoint(uri ...string) string {
	if n.User == nil || n.User.IsZero() {
		return ""
	}

	if len(uri) > 0 {
		return "/n/" + n.User.Username + "/" + n.Path + "/-/" + strings.Join(uri, "/")
	}
	return "/n/" + n.User.Username + "/" + n.Path
}

// Values implements the database.Model interface. This will return a map with
// the following values, user_id, root_id, parent_id, name, path, description,
// level, and visibility.
func (n *Namespace) Values() map[string]interface{} {
	if n == nil {
		return map[string]interface{}{}
	}
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

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either user.User, or namespace.Namespace models.
func (s *Store) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *user.User:
			s.User = v
		case *Namespace:
			s.Namespace = v
		}
	}
}

func (s *Store) getFromOwnerPath(path *string) (*Namespace, error) {
	u := s.User

	namespaces := NewStore(s.DB)

	if strings.Contains((*path), "@") {
		var err error

		parts := strings.Split((*path), "@")
		(*path) = parts[0]

		u, err = user.NewStore(s.DB).Get(query.Where("username", "=", query.Arg(parts[1])))

		if err != nil {
			return nil, errors.Err(err)
		}
	}

	n, err := namespaces.Get(database.Where(u, "user_id"), query.Where("path", "=", query.Arg(path)))

	if err != nil {
		return n, errors.Err(err)
	}

	if n.IsZero() {
		if !s.User.IsZero() {
			// Attempt was made to lookup a non-existent namespace for a user
			// other than the current one, return ErrPermission.
			if u.ID != s.User.ID {
				return nil, ErrPermission
			}
		}
		return n, nil
	}

	cc, err := NewCollaboratorStore(s.DB, n).All()

	if err != nil {
		return nil, errors.Err(err)
	}

	n.LoadCollaborators(cc)

	if !n.CanAdd(s.User) {
		return nil, ErrPermission
	}

	n.User = u
	s.User = u
	return n, nil
}

// Create creates a new namespace under the given parent, with the given name,
// description, and visibility. The parent and description variables can be
// empty.
func (s *Store) Create(parent, name, description string, visibility Visibility) (*Namespace, error) {
	p, err := s.Get(
		database.Where(s.User, "user_id"),
		query.Where("path", "=", query.Arg(parent)),
	)

	if err != nil {
		return nil, errors.Err(err)
	}

	if p.IsZero() && parent != "" {
		return nil, database.ErrNotFound
	}

	n := s.New()
	n.Name = name
	n.Path = name
	n.Description = description
	n.Visibility = visibility
	n.CreatedAt = time.Now()

	if !p.IsZero() {
		n.RootID = p.RootID
		n.ParentID = sql.NullInt64{
			Int64: p.ID,
			Valid: true,
		}
		n.Path = strings.Join([]string{p.Path, name}, "/")
		n.Level = p.Level + 1
		n.Visibility = p.Visibility
		n.Parent = p
	}

	if n.Level >= MaxDepth {
		return nil, ErrDepth
	}

	if err := s.Store.Create(table, n); err != nil {
		return nil, errors.Err(err)
	}

	if p.IsZero() {
		n.RootID = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}

		q := query.Update(
			table,
			query.Set("root_id", query.Arg(n.RootID)),
			query.Where("id", "=", query.Arg(n.ID)),
		)
		_, err := s.DB.Exec(q.Build(), q.Args()...)
		return n, errors.Err(err)
	}
	return n, nil
}

// Update updates the namespace of the given id with the given name, description
// and visibility. If the namespace is a root namespace, then the new visibility
// is applied to all children.
func (s *Store) Update(id int64, description string, visibility Visibility) error {
	parent, err := s.Get(
		query.Where("id", "=", query.Select(
			query.Columns("parent_id"),
			query.From(table),
			query.Where("id", "=", query.Arg(id)),
		)),
	)

	if err != nil {
		return errors.Err(err)
	}

	if !parent.IsZero() {
		visibility = parent.Visibility
	} else {
		q := query.Update(
			table,
			query.Set("visibility", query.Arg(visibility)),
			query.Where("root_id", "=", query.Arg(id)),
		)

		if _, err = s.DB.Exec(q.Build(), q.Args()...); err != nil {
			return errors.Err(err)
		}
	}

	q := query.Update(
		table,
		query.Set("description", query.Arg(description)),
		query.Set("visibility", query.Arg(visibility)),
		query.Where("id", "=", query.Arg(id)),
	)

	_, err = s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// Delete deletes the namespace of the given id, and all of their children.
func (s *Store) Delete(ids ...int64) error {
	vals := make([]interface{}, 0, len(ids))

	for _, id := range ids {
		vals = append(vals, id)
	}

	q := query.Delete(
		table,
		query.Where("id", "IN", query.List(vals...)),
		query.OrWhere("root_id", "IN", query.List(vals...)),
		query.OrWhere("parent_id", "IN", query.List(vals...)),
	)

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// Paginate returns the database.Paginator for the namespaces table for the given
// page. This applies the SharedWith option on the bound User database, and the
// database.Where option to the bound Namespace database on the "parent_id".
func (s Store) Paginate(page int64, opts ...query.Option) (database.Paginator, error) {
	paginator, err := s.Store.Paginate(table, page, 25, opts...)
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
// given. The SharedWith option is applied to the bound User database, and the
// database.Where option to the bound Namespace database on the "parent_id".
func (s *Store) All(opts ...query.Option) ([]*Namespace, error) {
	nn := make([]*Namespace, 0)

	opts = append([]query.Option{
		SharedWith(s.User),
		database.Where(s.Namespace, "parent_id"),
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
// path - This applies the database.Search query.Option using the value os path
func (s *Store) Index(vals url.Values, opts ...query.Option) ([]*Namespace, database.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		database.Search("path", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, opts...)

	if err != nil {
		return []*Namespace{}, paginator, errors.Err(err)
	}

	nn, err := s.All(append(
		opts,
		query.OrderAsc("path"),
		query.Limit(paginator.Limit),
		query.Offset(paginator.Offset),
	)...)
	return nn, paginator, errors.Err(err)
}

// Get returns a single Namespace database, applying each query.Option that is
// given. The SharedWith option is applied to the bound User database, and the
// database.Where option to the bound Namespace database on the "parent_id".
func (s *Store) Get(opts ...query.Option) (*Namespace, error) {
	n := &Namespace{
		User:   s.User,
		Parent: s.Namespace,
	}

	opts = append([]query.Option{
		database.Where(s.Namespace, "parent_id"),
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
	n, err := s.getFromOwnerPath(&path)

	if err != nil {
		return nil, errors.Err(err)
	}

	if !n.IsZero() {
		return n, nil
	}

	p := &Namespace{}

	parts := strings.Split(path, "/")

	for i, name := range parts {
		if p.Level+1 > MaxDepth {
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

		if err := s.Store.Create(table, n); err != nil {
			return n, errors.Err(err)
		}

		if p.IsZero() {
			n.RootID = sql.NullInt64{
				Int64: n.ID,
				Valid: true,
			}

			q := query.Update(
				table,
				query.Set("root_id", query.Arg(n.RootID)),
				query.Where("id", "=", query.Arg(n.ID)),
			)

			_, err = s.DB.Exec(q.Build(), q.Args()...)
			return n, errors.Err(err)
		}

		p = n
	}
	return n, nil
}

// Load loads in a slice of Namespace models where the given key is in the list
// of given vals. Each model is loaded individually via a call to the given
// load callback. This method calls Store.All under the hood, so any
// bound models will impact the models being loaded.
func (s *Store) Load(key string, vals []interface{}, load database.LoaderFunc) error {
	nn, err := s.All(query.Where(key, "IN", database.List(vals...)))

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
