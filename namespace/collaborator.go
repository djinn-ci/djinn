package namespace

import (
	"database/sql"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Collaborator is the type that represents a collaborator within a namespace.
type Collaborator struct {
	ID          int64     `db:"id"`
	NamespaceID int64     `db:"namespace_id"`
	UserID      int64     `db:"user_id"`
	CreatedAt   time.Time `db:"created_at"`

	User      *user.User `db:"-"`
	Namespace *Namespace `db:"-"`
}

// CollaboratorStore is the type for creating and modifying Collaborator models
// in the database.
type CollaboratorStore struct {
	database.Store

	// User is the bound User model. If not nil this will bind the User model to
	// any Collaborator models that are created. If not nil this will be passed
	// to the namespace.WhereCollaborator query option on each SELECT query
	// performed.
	User *user.User

	// Namespace is the bound Namespace model. If not nil this will bind the
	// Namespace model to any Collaborator models that are created. If not nil
	// this will append a WHERE clause on the namespace_id column for all SELECT
	// queries performed.
	Namespace *Namespace
}

var (
	_ database.Model  = (*Collaborator)(nil)
	_ database.Binder = (*CollaboratorStore)(nil)
	_ database.Loader = (*CollaboratorStore)(nil)

	collaboratorTable = "namespace_collaborators"
)

// NewCollaboratorStore returns a new Store for querying the
// namespace_collaborators table. Each database passed to this function will be
// bound to the returned Store.
func NewCollaboratorStore(db *sqlx.DB, mm ...database.Model) *CollaboratorStore {
	s := &CollaboratorStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// CollaboratorModel is called along with database.ModelSlice to convert the given
// slice of Collaborator models to a slice of database.Model interfaces.
func CollaboratorModel(cc []*Collaborator) func(int) database.Model {
	return func(i int) database.Model {
		return cc[i]
	}
}

// CollaboratorSelect returns a query that selects the given column from the
// namespace_collaborators table, with each given query.Option applied to the
// returned query.
func CollaboratorSelect(col string, opts ...query.Option) query.Query {
	return query.Select(query.Columns(col), append([]query.Option{query.From(collaboratorTable)}, opts...)...)
}

// WhereCollaborator returns a query.Option that when applied to a query will
// check to see if the given database.Model (assuming it's of type *user.User),
// is either a Collaborator, or owner of a Namespace. This would typically
// be used on resource queries, for example if you want to get a slice of
// *variable.Variable models for a User.
func WhereCollaborator(m database.Model) query.Option {
	return func(q query.Query) query.Query {
		if _, ok := m.(*user.User); !ok {
			return q
		}
		if m.IsZero() {
			return q
		}

		_, userId := m.Primary()

		return query.Options(
			query.Where("namespace_id", "IN",
				query.Select(
					query.Columns("id"),
					query.From(table),
					query.Where("root_id", "IN",
						query.Union(
							query.Select(
								query.Columns("namespace_id"),
								query.From(collaboratorTable),
								query.Where("user_id", "=", query.Arg(userId)),
							),
							query.Select(
								query.Columns("id"),
								query.From(table),
								query.Where("user_id", "=", query.Arg(userId)),
							),
						),
					),
				),
			),
			query.OrWhere("user_id", "=", query.Arg(userId)),
		)(q)
	}
}

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either user.User, or namespace.Namespace models.
func (c *Collaborator) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *Namespace:
			c.Namespace = v
		case *user.User:
			c.User = v
		}
	}
}

// SetPrimary implements the database.Model interface.
func (c *Collaborator) SetPrimary(id int64) { c.ID = id }

// Primary implements the database.Model interface.
func (c *Collaborator) Primary() (string, int64) { return "id", c.ID }

// IsZero implements the database.Model interface.
func (c *Collaborator) IsZero() bool {
	return c == nil || (c.ID == 0 && c.NamespaceID == 0 && c.UserID == 0)
}

// JSON implements the database.Model interface. If the bound User model is nil
// or zero then an empty map is returned, otherwise this will return the JSON
// representation of that bound User model, along with the url and created_at
// fields.
func (c *Collaborator) JSON(addr string) map[string]interface{} {
	if c.User == nil || c.User.IsZero() {
		return map[string]interface{}{}
	}

	json := c.User.JSON(addr)
	json["created_at"] = c.CreatedAt.Format(time.RFC3339)
	json["url"] = addr + c.Endpoint()
	return json
}

// Endpoint implements the database.Model interface. This will return an empty
// string if the bound Namespace, or User models are either nil or zero.
func (c *Collaborator) Endpoint(uri ...string) string {
	if c.Namespace == nil || c.Namespace.IsZero() {
		return ""
	}
	if c.User == nil || c.User.IsZero() {
		return ""
	}
	return c.Namespace.Endpoint(append(uri, "collaborators", c.User.Username)...)
}

// Values implements the database.Model interface. This will return a map with
// the following values, namespace_id and user_id.
func (c *Collaborator) Values() map[string]interface{} {
	return map[string]interface{}{
		"namespace_id": c.NamespaceID,
		"user_id":      c.UserID,
	}
}

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either user.User, or namespace.Namespace models.
func (s *CollaboratorStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *Namespace:
			s.Namespace = v
		case *user.User:
			s.User = v
		}
	}
}

// Create inserts the given Collaborator models into the
// namespace_collaborators table.
func (s *CollaboratorStore) Create(cc ...*Collaborator) error {
	models := database.ModelSlice(len(cc), CollaboratorModel(cc))
	return s.Store.Create(collaboratorTable, models...)
}

// Delete will remove the collaborator of the given username from the database.
// The given chown functions will be invoked to change ownership of any
// resources that user may have created within the namespace to the owner of
// the namespace. This requires a bound Namespace to execute successfully.
func (s *CollaboratorStore) Delete(username string, chowns ...func(int64, int64) error) error {
	if s.Namespace == nil {
		return errors.New("nil bound namespace")
	}

	u, err := user.NewStore(s.DB).Get(query.Where("username", "=", query.Arg(username)))

	if err != nil {
		return errors.Err(err)
	}

	q := query.Delete(
		collaboratorTable,
		query.Where("user_id", "=", user.Select("id", query.Where("username", "=", query.Arg(username)))),
	)

	if _, err := s.DB.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}

	for _, chown := range chowns {
		if err := chown(u.ID, s.Namespace.UserID); err != nil {
			return errors.Err(err)
		}
	}
	return nil
}

// New returns a new Collaborator binding any non-nil models to it from the
// current CollaboratorStore.
func (s *CollaboratorStore) New() *Collaborator {
	c := &Collaborator{
		Namespace: s.Namespace,
		User:      s.User,
	}

	if s.Namespace != nil {
		c.NamespaceID = s.Namespace.ID
	}

	if s.User != nil {
		c.UserID = s.User.ID
	}
	return c
}

// All returns a slice of Collaborator models, applying each query.Option that
// is given. The database.Where option is applied to the bound User database, and the
// bound Namespace database using the "root_id" column as the value to perform the
// WHERE clause on.
func (s *CollaboratorStore) All(opts ...query.Option) ([]*Collaborator, error) {
	cc := make([]*Collaborator, 0)

	opts = append([]query.Option{
		database.Where(s.User, "user_id"),
		database.Where(s.Namespace, "namespace_id", "root_id"),
	}, opts...)

	err := s.Store.All(&cc, collaboratorTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, c := range cc {
		c.User = s.User
		c.Namespace = s.Namespace
	}
	return cc, errors.Err(err)
}

// Get returns a single Collaborator database, applying each query.Option that
// is given. The database.Where option is applied to the bound User database, and the
// bound Namespace database using the "root_id" column as the value to perform the
// WHERE clause on.
func (s *CollaboratorStore) Get(opts ...query.Option) (*Collaborator, error) {
	c := &Collaborator{
		Namespace: s.Namespace,
		User:      s.User,
	}

	opts = append([]query.Option{
		database.Where(s.User, "user_id"),
		database.Where(s.Namespace, "namespace_id", "root_id"),
	}, opts...)

	err := s.Store.Get(c, collaboratorTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return c, errors.Err(err)
}

// Load loads in a slice of Collaborator models where the given key is in the
// list of given vals. Each database is loaded individually via a call to the given
// load callback. This method calls CollaboratorStore.All under the hood, so any
// bound models will impact the models being loaded.
func (s *CollaboratorStore) Load(key string, vals []interface{}, load database.LoaderFunc) error {
	cc, err := s.All(query.Where(key, "IN", database.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, c := range cc {
			load(i, c)
		}
	}
	return nil
}
