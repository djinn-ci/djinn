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

type Invite struct {
	ID          int64     `db:"id"`
	NamespaceID int64     `db:"namespace_id"`
	InviteeID   int64     `db:"invitee_id"`
	InviterID   int64     `db:"inviter_id"`
	CreatedAt   time.Time `db:"created_at"`

	User      model.Model `db:"-"`
	Namespace *Namespace  `db:"-"`
}

type InviteStore struct {
	model.Store

	User      model.Model
	Namespace *Namespace
}

var (
	_ model.Model  = (*Invite)(nil)
	_ model.Binder = (*InviteStore)(nil)

	inviteTable     = "namespace_invites"
	inviteRelations = map[string]model.RelationFunc{
		"namespace": model.Relation("namespace_id", "id"),
		"user":      model.Relation("inviter_id", "id"),
	}
)

func NewInviteStore(db *sqlx.DB, mm ...model.Model) InviteStore {
	s := InviteStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func LoadInviteRelations(loaders model.Loaders, ii ...*Invite) error {
	mm := model.Slice(len(ii), InviteModel(ii))
	return errors.Err(model.LoadRelations(inviteRelations, loaders, mm...))
}

func InviteModel(ii []*Invite) func(int) model.Model {
	return func(i int) model.Model {
		return ii[i]
	}
}

func (i *Invite) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.Kind() {
		case "user":
			i.User = m
		case "namespace":
			i.Namespace = m.(*Namespace)
		}
	}
}

func (i Invite) Kind() string { return "namespace_invites" }

func (i *Invite) SetPrimary(id int64) {
	i.ID = id
}

func (i Invite) Primary() (string, int64) {
	return "id", i.ID
}

func (i Invite) Endpoint(_ ...string) string {
	return fmt.Sprintf("/invites/%v", i.ID)
}

func (i *Invite) IsZero() bool {
	return i == nil || i.ID == 0 &&
		i.NamespaceID == 0 &&
		i.InviteeID == 0 &&
		i.InviterID == 0 &&
		i.CreatedAt == time.Time{}
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

	opts = append([]query.Option{
		model.Where(s.User, "invitee_id"),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.All(&ii, inviteTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return ii, errors.Err(err)
}

func (s InviteStore) Get(opts ...query.Option) (*Invite, error) {
	i := &Invite{
		User:      s.User,
		Namespace: s.Namespace,
	}

	opts = append([]query.Option{
		model.Where(s.User, "invitee_id"),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.Get(i, inviteTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return i, errors.Err(err)
}

func (s InviteStore) Create(ii ...*Invite) error {
	mm := model.Slice(len(ii), InviteModel(ii))
	return errors.Err(s.Store.Create(inviteTable, mm...))
}

func (s InviteStore) Delete(ii ...*Invite) error {
	mm := model.Slice(len(ii), InviteModel(ii))
	return errors.Err(s.Store.Delete(inviteTable, mm...))
}

func (s InviteStore) New() *Invite {
	i := &Invite{
		Namespace: s.Namespace,
	}

	if s.Namespace != nil {
		i.NamespaceID = s.Namespace.ID
	}
	return i
}

func (s *InviteStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.Kind() {
		case "user":
			s.User = m
		case "namespace":
			s.Namespace = m.(*Namespace)
		}
	}
}
