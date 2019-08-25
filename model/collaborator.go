package model

import (
	"database/sql"
	"fmt"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"
)

type Collaborator struct {
	Model

	NamespaceID	int64 `db:"namespace_id"`
	UserID      int64 `db:"user_id"`

	Namespace *Namespace
	User      *User
}

type CollaboratorStore struct {
	Store

	Namespace *Namespace
	User      *User
}

func collaboratorToInterface(cc ...*Collaborator) func(i int) Interface {
	return func(i int) Interface {
		return cc[i]
	}
}

func (c Collaborator) IsZero() bool {
	return c.Model.IsZero() &&
		c.NamespaceID == 0 &&
		c.UserID == 0
}

func (c Collaborator) UIEndpoint(uris ...string) string {
	if c.Namespace == nil || c.Namespace.IsZero() {
		return ""
	}

	uris = append(uris, "-", "collaborators", fmt.Sprintf("%v", c.ID))

	return c.Namespace.UIEndpoint(uris...)
}

func (c Collaborator) Values() map[string]interface{} {
	return map[string]interface{}{
		"namespace_id": c.NamespaceID,
		"user_id":      c.UserID,
	}
}

func (s CollaboratorStore) All(opts ...query.Option) ([]*Collaborator, error) {
	cc := make([]*Collaborator, 0)

	opts = append(opts, ForRootNamespace(s.Namespace), ForUser(s.User))

	err := s.Store.All(&cc, CollaboratorTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, c := range cc {
		c.DB = s.DB
		c.Namespace = s.Namespace
		c.User = s.User
	}

	return cc, errors.Err(err)
}

func (s CollaboratorStore) Index(opts ...query.Option) ([]*Collaborator, error) {
	cc, err := s.All(opts...)

	if err != nil {
		return cc, errors.Err(err)
	}

	err = s.LoadUsers(cc)

	return cc, errors.Err(err)
}

func (s CollaboratorStore) loadUser(cc []*Collaborator) func(i int, u *User) {
	return func(i int, u *User) {
		c := cc[i]

		if c.UserID == u.ID {
			c.User = u
		}
	}
}

func (s CollaboratorStore) LoadUsers(cc []*Collaborator) error {
	if len(cc) == 0 {
		return nil
	}
	models := interfaceSlice(len(cc), collaboratorToInterface(cc...))

	users := UserStore{
		Store: s.Store,
	}

	err := users.Load(mapKey("user_id", models), s.loadUser(cc))

	return errors.Err(err)
}

func (s CollaboratorStore) Create(cc ...*Collaborator) error {
	models := interfaceSlice(len(cc), collaboratorToInterface(cc...))

	return errors.Err(s.Store.Create(CollaboratorTable, models...))
}

func (s CollaboratorStore) Delete(cc ...*Collaborator) error {
	models := interfaceSlice(len(cc), collaboratorToInterface(cc...))

	return errors.Err(s.Store.Delete(CollaboratorTable, models...))
}

func (c *Collaborator) LoadUser() error {
	var err error

	user := UserStore{
		Store: Store{
			DB: c.DB,
		},
	}

	c.User, err = user.Find(c.UserID)

	return errors.Err(err)
}

func (s CollaboratorStore) FindByHandle(handle string) (*Collaborator, error) {
	c := &Collaborator{
		Model: Model{
			DB: s.DB,
		},
		Namespace: s.Namespace,
		User:      s.User,
	}

	q := query.Select(
		query.Columns("*"),
		query.Table(CollaboratorTable),
		query.WhereEqQuery("user_id",
			query.Select(
				query.Columns("id"),
				query.Table(UserTable),
				query.Or(
					query.WhereEq("email", handle),
					query.WhereEq("username", handle),
				),
			),
		),
		ForNamespace(s.Namespace),
	)

	err := s.Store.Get(c, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return c, errors.Err(err)
}

func (s CollaboratorStore) New() *Collaborator {
	c := &Collaborator{
		Model: Model{
			DB: s.DB,
		},
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
