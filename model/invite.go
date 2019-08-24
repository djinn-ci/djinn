package model

import (
	"database/sql"
	"fmt"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"
)

type Invite struct {
	Model

	NamespaceID int64 `db:"namespace_id"`
	InviteeID   int64 `db:"invitee_id"`
	InviterID   int64 `db:"inviter_id"`

	Namespace *Namespace
	Invitee   *User
	Inviter   *User
}

type InviteStore struct {
	Store

	Namespace *Namespace
}

func (i *Invite) LoadNamespace() error {
	var err error

	namespaces := NamespaceStore{
		Store: Store{
			DB: i.DB,
		},
	}

	i.Namespace, err = namespaces.Find(i.NamespaceID)

	return errors.Err(err)
}

func (i Invite) UIEndpoint() string {
	return fmt.Sprintf("/invites/%v", i.ID)
}

func (i Invite) Values() map[string]interface{} {
	return map[string]interface{}{
		"namespace_id": i.NamespaceID,
		"invitee_id":   i.InviteeID,
		"inviter_id":   i.InviterID,
	}
}

func (s InviteStore) All(opts ...query.Option) ([]*Invite, error) {
	ii := make([]*Invite, 0)

	opts = append(opts, ForNamespace(s.Namespace))

	err := s.Store.All(&ii, InviteTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, i := range ii {
		i.DB = s.DB
		i.Namespace = s.Namespace
	}

	return ii, errors.Err(err)
}

func (s InviteStore) Create(ii ...*Invite) error {
	return errors.Err(s.Store.Create(InviteTable, s.interfaceSlice(ii...)...))
}

func (s InviteStore) Delete(ii ...*Invite) error {
	return errors.Err(s.Store.Delete(InviteTable, s.interfaceSlice(ii...)...))
}

func (s InviteStore) Find(id int64) (*Invite, error) {
	i := &Invite{
		Model: Model{
			DB: s.DB,
		},
		Namespace: s.Namespace,
	}

	err := s.Store.FindBy(i, InviteTable, "id", id)

	if err == sql.ErrNoRows {
		err = nil
	}

	return i, errors.Err(err)
}

func (s InviteStore) FindByHandle(handle string) (*Invite, error) {
	i := &Invite{
		Model: Model{
			DB: s.DB,
		},
		Namespace: s.Namespace,
	}

	q := query.Select(
		query.Columns("*"),
		query.Table(InviteTable),
		query.WhereEqQuery("invitee_id",
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

	err := s.Store.Get(i, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return i, errors.Err(err)
}

func (s InviteStore) interfaceSlice(invites ...*Invite) []Interface {
	ii := make([]Interface, len(invites), len(invites))

	for i, inv := range invites {
		ii[i] = inv
	}

	return ii
}

func loadInviteInviter(ii []*Invite) func(i int, u *User) {
	return func(i int, u *User) {
		inv := ii[i]

		if inv.InviterID == u.ID {
			inv.Inviter = u
		}
	}
}

func loadInviteNamespace(ii []*Invite) func(i int, n *Namespace) {
	return func(i int, n *Namespace) {
		inv := ii[i]

		if inv.NamespaceID == n.ID {
			inv.Namespace = n
		}
	}
}

func (s InviteStore) Index(opts ...query.Option) ([]*Invite, error) {
	ii, err := s.All(opts...)

	if err != nil {
		return ii, errors.Err(err)
	}

	users := UserStore{
		Store: s.Store,
	}

	err = users.Load(mapKey("inviter_id", s.interfaceSlice(ii...)), loadInviteInviter(ii))

	namespaces := NamespaceStore{
		Store: s.Store,
	}

	err = namespaces.Load(mapKey("namespace_id", s.interfaceSlice(ii...)), loadInviteNamespace(ii))

	return ii, errors.Err(err)
}

func (s InviteStore) New() *Invite {
	i := &Invite{
		Model: Model{
			DB: s.DB,
		},
		Namespace: s.Namespace,
	}

	if s.Namespace != nil {
		i.NamespaceID = s.Namespace.ID
	}

	return i
}
