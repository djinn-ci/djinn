package namespace

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type Collaborator struct {
	ID          int64     `db:"id"`
	NamespaceID int64     `db:"namespace_id"`
	UserID      int64     `db:"user_id"`
	CreatedAt   time.Time `db:"created_at"`

	User      model.Model `db:"-"`
	Namespace *Namespace  `db:"-"`
}

type CollaboratorStore struct {
	model.Store

	User      model.Model
	Namespace *Namespace
}

var (
	_ model.Model  = (*Collaborator)(nil)
	_ model.Binder = (*CollaboratorStore)(nil)
	_ model.Loader = (*CollaboratorStore)(nil)

	collaboratorTable = "namespace_collaborators"
)

func NewCollaboratorStore(db *sqlx.DB, mm ...model.Model) CollaboratorStore {
	s := CollaboratorStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func CollaboratorModel(cc []*Collaborator) func(int) model.Model {
	return func(i int) model.Model {
		return cc[i]
	}
}

func WhereCollaborator(m model.Model) query.Option {
	return func(q query.Query) query.Query {
		if m.IsZero() || m.Kind() != "user" {
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

func (c *Collaborator) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.Kind() {
		case "namespace":
			c.Namespace = m.(*Namespace)
		case "user":
			c.User = m
		}
	}
}

func (c Collaborator) Kind() string { return "namespace_collaborator "}

func (c *Collaborator) SetPrimary(id int64) {
	c.ID = id
}

func (c Collaborator) Primary() (string, int64) {
	return "id", c.ID
}

func (c Collaborator) IsZero() bool {
	return c.ID == 0 && c.NamespaceID == 0 && c.UserID == 0
}

func (c Collaborator) Endpoint(uri ...string) string {
	if c.Namespace == nil || c.Namespace.IsZero() {
		return ""
	}
	if c.User == nil || c.User.IsZero() {
		return ""
	}

	username, _ := c.User.Values()["username"].(string)

	uri = append(uri, "-", "collaborators", fmt.Sprintf("%s", username))
	return c.Namespace.Endpoint(uri...)
}

func (c Collaborator) Values() map[string]interface{} {
	return map[string]interface{}{
		"namespace_id": c.NamespaceID,
		"user_id":      c.UserID,
	}
}

func (s *CollaboratorStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.Kind() {
		case "namespace":
			s.Namespace = m.(*Namespace)
		case "user":
			s.User = m
		}
	}
}

func (s CollaboratorStore) Create(cc ...*Collaborator) error {
	models := model.Slice(len(cc), CollaboratorModel(cc))
	return s.Store.Create(collaboratorTable, models...)
}

func (s CollaboratorStore) Delete(cc ...*Collaborator) error {
	models := model.Slice(len(cc), CollaboratorModel(cc))
	return s.Store.Delete(collaboratorTable, models...)
}

func (s CollaboratorStore) New() *Collaborator {
	c := &Collaborator{
		Namespace: s.Namespace,
		User:      s.User,
	}

	if s.Namespace != nil {
		_, id := s.Namespace.Primary()
		c.NamespaceID = id
	}

	if s.User != nil {
		_, id := s.Namespace.Primary()
		c.UserID = id
	}
	return c
}

func (s CollaboratorStore) All(opts ...query.Option) ([]*Collaborator, error) {
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

func (s CollaboratorStore) Get(opts ...query.Option) (*Collaborator, error) {
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

func (s CollaboratorStore) Load(key string, vals []interface{}, load model.LoaderFunc) error {
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
