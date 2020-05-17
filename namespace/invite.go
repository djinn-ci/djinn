package namespace

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type Invite struct {
	ID          int64     `db:"id"`
	NamespaceID int64     `db:"namespace_id"`
	InviteeID   int64     `db:"invitee_id"`
	InviterID   int64     `db:"inviter_id"`
	CreatedAt   time.Time `db:"created_at"`

	Inviter   *user.User `db:"-"`
	Namespace *Namespace `db:"-"`
}

type InviteStore struct {
	model.Store

	User      *user.User
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

// NewInviteStore returns a new Store for querying the namespace_invites table.
// Each model passed to this function will be bound to the returned Store.
func NewInviteStore(db *sqlx.DB, mm ...model.Model) *InviteStore {
	s := &InviteStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// LoadInviteRelations loads all of the available relations for the given
// Invite models using the given loaders available.
func LoadInviteRelations(loaders model.Loaders, ii ...*Invite) error {
	mm := model.Slice(len(ii), InviteModel(ii))
	return errors.Err(model.LoadRelations(inviteRelations, loaders, mm...))
}

// InviteModel is called along with model.Slice to convert the given slice of
// Invite models to a slice of model.Model interfaces.
func InviteModel(ii []*Invite) func(int) model.Model {
	return func(i int) model.Model {
		return ii[i]
	}
}

// Bind the given models to the current Invite. This will only bind the
// model if they are one of the following,
//
// - *user.User
// - *namespace.Namespace
//
// the User model is bound to the Inviter field.
func (i *Invite) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			i.Inviter = m.(*user.User)
		case *Namespace:
			i.Namespace = m.(*Namespace)
		}
	}
}

func (i *Invite) SetPrimary(id int64) {
	i.ID = id
}

func (i *Invite) Primary() (string, int64) { return "id", i.ID }

// Endpoint returns the endpoint for the current Invite. This does not append
// any of the given URIs.
func (i *Invite) Endpoint(_ ...string) string { return fmt.Sprintf("/invites/%v", i.ID) }

func (i *Invite) IsZero() bool {
	return i == nil || i.ID == 0 &&
		i.NamespaceID == 0 &&
		i.InviteeID == 0 &&
		i.InviterID == 0 &&
		i.CreatedAt == time.Time{}
}

func (i *Invite) Values() map[string]interface{} {
	return map[string]interface{}{
		"namespace_id": i.NamespaceID,
		"invitee_id":   i.InviteeID,
		"inviter_id":   i.InviterID,
	}
}

// All returns a slice of Invite models, applying each query.Option that is
// given. The model.Where option is applied to the bound User model, and the
// bound Namespace model.
func (s *InviteStore) All(opts ...query.Option) ([]*Invite, error) {
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

// Get returns a single Invite model, applying each query.Option that is given.
// The model.Where option is applied to the bound User model, and the bound
// Namespace model.
func (s *InviteStore) Get(opts ...query.Option) (*Invite, error) {
	i := &Invite{
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

// Create inserts the given Invite models into the namespace_invites table.
func (s *InviteStore) Create(ii ...*Invite) error {
	mm := model.Slice(len(ii), InviteModel(ii))
	return errors.Err(s.Store.Create(inviteTable, mm...))
}

// Update updates the given Invite models in the namespace_invites table.
func (s *InviteStore) Delete(ii ...*Invite) error {
	mm := model.Slice(len(ii), InviteModel(ii))
	return errors.Err(s.Store.Delete(inviteTable, mm...))
}

// New returns a new Invite binding any non-nil models to it from the current
// Invite.
func (s *InviteStore) New() *Invite {
	i := &Invite{
		Namespace: s.Namespace,
	}

	if s.Namespace != nil {
		i.NamespaceID = s.Namespace.ID
	}
	return i
}

// Bind the given models to the current InviteStore. This will only bind the
// model if they are one of the following,
//
// - *user.User
// - *namespace.Namespace
func (s *InviteStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		case *Namespace:
			s.Namespace = m.(*Namespace)
		}
	}
}
