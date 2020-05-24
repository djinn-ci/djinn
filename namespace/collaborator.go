package namespace

import (
	"database/sql"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type Collaborator struct {
	ID          int64     `db:"id"`
	NamespaceID int64     `db:"namespace_id"`
	UserID      int64     `db:"user_id"`
	CreatedAt   time.Time `db:"created_at"`

	User      *user.User `db:"-"`
	Namespace *Namespace `db:"-"`
}

type CollaboratorStore struct {
	model.Store

	User      *user.User
	Namespace *Namespace
}

var (
	_ model.Model  = (*Collaborator)(nil)
	_ model.Binder = (*CollaboratorStore)(nil)
	_ model.Loader = (*CollaboratorStore)(nil)

	collaboratorTable = "namespace_collaborators"
)

// NewCollaboratorStore returns a new Store for querying the
// namespace_collaborators table. Each model passed to this function will be
// bound to the returned Store.
func NewCollaboratorStore(db *sqlx.DB, mm ...model.Model) *CollaboratorStore {
	s := &CollaboratorStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// CollaboratorModel is called along with model.Slice to convert the given
// slice of Collaborator models to a slice of model.Model interfaces.
func CollaboratorModel(cc []*Collaborator) func(int) model.Model {
	return func(i int) model.Model {
		return cc[i]
	}
}

// WhereCollaborator returns a query.Option that when applied to a query will
// check to see if the given model.Model (assuming it's of type *user.User),
// is either a Collaborator, or owner of a Namespace. This would typically
// be used on resource queries, for example if you want to get a slice of
// *variable.Variable models for a User.
func WhereCollaborator(m model.Model) query.Option {
	return func(q query.Query) query.Query {
		if _, ok := m.(*user.User); !ok {
			return q
		}
		if m.IsZero() {
			return q
		}

		_, userId := m.Primary()

		return query.Options(
			query.WhereQuery("namespace_id", "IN",
				query.Select(
					query.Columns("id"),
					query.From(table),
					query.WhereQuery("root_id", "IN",
						query.Union(
							query.Select(
								query.Columns("namespace_id"),
								query.From(collaboratorTable),
								query.Where("user_id", "=", userId),
							),
							query.Select(
								query.Columns("id"),
								query.From(table),
								query.Where("user_id", "=", userId),
							),
						),
					),
				),
			),
			query.OrWhere("user_id", "=", userId),
		)(q)
	}
}

// Bind the given models to the current Collaborator. This will only bind the
// model if they are one of the following,
//
// - *user.User
// - *namespace.Namespace
func (c *Collaborator) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Namespace:
			c.Namespace = m.(*Namespace)
		case *user.User:
			c.User = m.(*user.User)
		}
	}
}

func (c *Collaborator) SetPrimary(id int64) {
	c.ID = id
}

func (c *Collaborator) Primary() (string, int64) {
	return "id", c.ID
}

func (c *Collaborator) IsZero() bool {
	return c == nil || (c.ID == 0 && c.NamespaceID == 0 && c.UserID == 0)
}

func (c *Collaborator) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":            c.ID,
		"namespace_id":  c.NamespaceID,
		"user_id":       c.UserID,
		"created_at":    c.CreatedAt.Format(time.RFC3339),
		"url":           addr + c.Endpoint(),
	}

	for name, m := range map[string]model.Model{
		"user":      c.User,
		"namespace": c.Namespace,
	}{
		if !m.IsZero() {
			json[name] = m.JSON(addr)
		}
	}
	return json
}

func (c *Collaborator) Endpoint(uri ...string) string {
	if c.Namespace == nil || c.Namespace.IsZero() {
		return ""
	}
	if c.User == nil || c.User.IsZero() {
		return ""
	}
	return c.Namespace.Endpoint(append(uri, "-", "collaborators", c.User.Username)...)
}

func (c *Collaborator) Values() map[string]interface{} {
	return map[string]interface{}{
		"namespace_id": c.NamespaceID,
		"user_id":      c.UserID,
	}
}

// Bind the given models to the current CollaboratorStore. This will only bind
// the model if they are one of the following,
//
// - *user.User
// - *namespace.Namespace
func (s *CollaboratorStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Namespace:
			s.Namespace = m.(*Namespace)
		case *user.User:
			s.User = m.(*user.User)
		}
	}
}

// Create inserts the given Collaborator models into the
// namespace_collaborators table.
func (s *CollaboratorStore) Create(cc ...*Collaborator) error {
	models := model.Slice(len(cc), CollaboratorModel(cc))
	return s.Store.Create(collaboratorTable, models...)
}

// Delete delets the given Collaborator models from the
// namespace_collaborators table.
func (s *CollaboratorStore) Delete(cc ...*Collaborator) error {
	models := model.Slice(len(cc), CollaboratorModel(cc))
	return s.Store.Delete(collaboratorTable, models...)
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
// is given. The model.Where option is applied to the bound User model, and the
// bound Namespace model using the "root_id" column as the value to perform the
// WHERE clause on.
func (s *CollaboratorStore) All(opts ...query.Option) ([]*Collaborator, error) {
	cc := make([]*Collaborator, 0)

	opts = append([]query.Option{
		model.Where(s.User, "user_id"),
		model.Where(s.Namespace, "namespace_id", "root_id"),
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

// Get returns a single Collaborator model, applying each query.Option that
// is given. The model.Where option is applied to the bound User model, and the
// bound Namespace model using the "root_id" column as the value to perform the
// WHERE clause on.
func (s *CollaboratorStore) Get(opts ...query.Option) (*Collaborator, error) {
	c := &Collaborator{
		Namespace: s.Namespace,
		User:      s.User,
	}

	opts = append([]query.Option{
		model.Where(s.User, "user_id"),
		model.Where(s.Namespace, "namespace_id", "root_id"),
	}, opts ...)

	err := s.Store.Get(c, collaboratorTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return c, errors.Err(err)
}

// Load loads in a slice of Collaborator models where the given key is in the
// list of given vals. Each model is loaded individually via a call to the given
// load callback. This method calls CollaboratorStore.All under the hood, so any
// bound models will impact the models being loaded.
func (s *CollaboratorStore) Load(key string, vals []interface{}, load model.LoaderFunc) error {
	cc, err := s.All(query.Where(key, "IN", vals...))

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
