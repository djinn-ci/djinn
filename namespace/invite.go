package namespace

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Invite is the type that represents an invite sent to a user for a namespace.
type Invite struct {
	ID          int64     `db:"id"`
	NamespaceID int64     `db:"namespace_id"`
	InviteeID   int64     `db:"invitee_id"`
	InviterID   int64     `db:"inviter_id"`
	CreatedAt   time.Time `db:"created_at"`

	Inviter   *user.User `db:"-"`
	Invitee   *user.User `db:"-"`
	Namespace *Namespace `db:"-"`
}

// InviteStore is the type for creating and mofiying Invite models in the
// database.
type InviteStore struct {
	database.Store

	// User is the bound User model. If not nil this will bind the User model to
	// any Invite models that are created. If not nil this will be passed to the
	// namespace.WhereCollaborator query option on each SELECT query performed.
	User *user.User

	// Namespace is the bound Namespace model. If not nil this will bind the
	// Namespace model to any Invite models that are created. If not nil this
	// will append a WHERE clause on the namespace_id column for all SELECT
	// queries performed.
	Namespace *Namespace
}

var (
	_ database.Model  = (*Invite)(nil)
	_ database.Binder = (*InviteStore)(nil)

	inviteTable     = "namespace_invites"
	inviteRelations = map[string]database.RelationFunc{
		"namespace": database.Relation("namespace_id", "id"),
		"user":      database.Relation("inviter_id", "id"),
		"inviter":   database.Relation("inviter_id", "id"),
		"invitee":   database.Relation("invitee_id", "id"),
	}
)

// NewInviteStore returns a new Store for querying the namespace_invites table.
// Each database passed to this function will be bound to the returned Store.
func NewInviteStore(db *sqlx.DB, mm ...database.Model) *InviteStore {
	s := &InviteStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// InviteFromContext returns the Invite model from the given context, if any.
func InviteFromContext(ctx context.Context) (*Invite, bool) {
	i, ok := ctx.Value("invite").(*Invite)
	return i, ok
}

// LoadInviteRelations loads all of the available relations for the given
// Invite models using the given loaders available.
func LoadInviteRelations(loaders *database.Loaders, ii ...*Invite) error {
	mm := database.ModelSlice(len(ii), InviteModel(ii))
	return errors.Err(database.LoadRelations(inviteRelations, loaders, mm...))
}

// InviteModel is called along with database.ModelSlice to convert the given slice of
// Invite models to a slice of database.Model interfaces.
func InviteModel(ii []*Invite) func(int) database.Model {
	return func(i int) database.Model {
		return ii[i]
	}
}

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either user.User, or namespace.Namespace models. The
// ID of the User model is checked agains the InviterID, and InviteeID of the
// current invite to determine if the User model should be bound as an Inviter
// or Invitee.
func (i *Invite) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			u := m.(*user.User)

			if u.ID == i.InviterID {
				i.Inviter = u
			}

			if u.ID == i.InviteeID {
				i.Invitee = u
			}
		case *Namespace:
			i.Namespace = m.(*Namespace)
		}
	}
}

// SetPrimary implements the database.Model interface.
func (i *Invite) SetPrimary(id int64) { i.ID = id }

// Primary implements the database.Model interface.
func (i *Invite) Primary() (string, int64) { return "id", i.ID }

// Endpoint returns the endpoint for the current Invite. This does not append
// any of the given URIs.
func (i *Invite) Endpoint(_ ...string) string { return "/invites/" + strconv.FormatInt(i.ID, 10) }

// IsZero implements the database.Model interface.
func (i *Invite) IsZero() bool {
	return i == nil || i.ID == 0 &&
		i.NamespaceID == 0 &&
		i.InviteeID == 0 &&
		i.InviterID == 0 &&
		i.CreatedAt == time.Time{}
}

// JSON implements the database.Model interface. If the bound User model is
// present as either invitee, or inviter then the JSON representation of this
// model will be under the invitee, or inviter keys respectively. If the
// Namespace bound model is present then the JSON representation of that model
// will be under the namespace key.
func (i *Invite) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":           i.ID,
		"namespace_id": i.NamespaceID,
		"invitee_id":   i.InviteeID,
		"inviter_id":   i.InviterID,
		"url":          addr + i.Endpoint(),
	}

	for name, m := range map[string]database.Model{
		"invitee":   i.Invitee,
		"inviter":   i.Inviter,
		"namespace": i.Namespace,
	}{
		if !m.IsZero() {
			json[name] = m.JSON(addr)
		}
	}
	return json
}

// Values implements the database.Model interface. This will return a map with
// the following values, namespace_id, invitee_id, and inviter_id.
func (i *Invite) Values() map[string]interface{} {
	return map[string]interface{}{
		"namespace_id": i.NamespaceID,
		"invitee_id":   i.InviteeID,
		"inviter_id":   i.InviterID,
	}
}

// All returns a slice of Invite models, applying each query.Option that is
// given. The database.Where option is applied to the bound User database, and the
// bound Namespace database.
func (s *InviteStore) All(opts ...query.Option) ([]*Invite, error) {
	ii := make([]*Invite, 0)

	opts = append([]query.Option{
		database.Where(s.User, "invitee_id"),
		database.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.All(&ii, inviteTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return ii, errors.Err(err)
}

// Get returns a single Invite database, applying each query.Option that is given.
// The database.Where option is applied to the bound User database, and the bound
// Namespace database.
func (s *InviteStore) Get(opts ...query.Option) (*Invite, error) {
	i := &Invite{
		Namespace: s.Namespace,
	}

	opts = append([]query.Option{
		database.Where(s.User, "invitee_id"),
		database.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.Get(i, inviteTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return i, errors.Err(err)
}

// Accept will delete the invite of the given ID, and create a new collaborator
// for the invited user. The namespace that the invite was for will be returned
// upon success.
func (s *InviteStore) Accept(id int64) (*Namespace, *user.User, *user.User, error) {
	i, err := s.Get(query.Where("id", "=", id))

	if err != nil {
		return nil, nil, nil, errors.Err(err)
	}

	uu, err := user.NewStore(s.DB).All(
		query.Where("id", "=", i.InviterID),
		query.OrWhere("id", "=", i.InviteeID),
	)

	var (
		inviter *user.User
		invitee *user.User
	)

	if uu[0].ID == i.InviterID {
		inviter = uu[0]
		invitee = uu[1]
	} else {
		inviter = uu[1]
		invitee = uu[0]
	}

	if err != nil {
		return nil, inviter, invitee, errors.Err(err)
	}

	n, err := NewStore(s.DB).Get(query.Where("id", "=", i.NamespaceID))

	if err != nil {
		return n, inviter, invitee, errors.Err(err)
	}

	collaborators := NewCollaboratorStore(s.DB, n)

	c := collaborators.New()
	c.UserID = i.InviteeID

	if err := collaborators.Create(c); err != nil {
		return n, inviter, invitee, errors.Err(err)
	}

	err = s.Delete(i)
	return n, inviter, invitee, errors.Err(err)
}

// Create will send an invite to the user specified via inviteeId. It is
// expected for the inviterId to match the user ID of the namespace's owner.
// The newly create invite will be returned.
func (s *InviteStore) Create(inviterId, inviteeId int64) (*Invite, error) {
	if inviterId != s.Namespace.UserID {
		return nil, ErrPermission
	}

	i := s.New()
	i.InviterID = inviterId
	i.InviteeID = inviteeId

	err := s.Store.Create(inviteTable, i)
	return i, errors.Err(err)
}

// Update updates the given Invite models in the namespace_invites table.
func (s *InviteStore) Delete(ii ...*Invite) error {
	mm := database.ModelSlice(len(ii), InviteModel(ii))
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

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either user.User, or namespace.Namespace models.
func (s *InviteStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		case *Namespace:
			s.Namespace = m.(*Namespace)
		}
	}
}
