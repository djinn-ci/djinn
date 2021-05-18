package handler

import (
	"fmt"
	"net/http"

	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/mail"
	"djinn-ci.com/namespace"
	"djinn-ci.com/user"
	"djinn-ci.com/web"

	"github.com/andrewpillar/webutil"

	"github.com/gorilla/mux"
)

// Invite is the base handler that provides shared logic for the UI and API
// handlers for working with build namespace invites.
type Invite struct {
	web.Handler

	loaders *database.Loaders
}

var inviteMail = `%s has invited you to be a collaborator in %s. You can accept this invite via
your Invites list,

    %s/invites`

func NewInvite(h web.Handler) Invite {
	loaders := database.NewLoaders()
	loaders.Put("user", h.Users)
	loaders.Put("inviter", h.Users)
	loaders.Put("invitee", h.Users)
	loaders.Put("namespace", namespace.NewStore(h.DB))

	return Invite{
		Handler: h,
		loaders: loaders,
	}
}

// IndexWithRelations returns all of the namespace invites with their
// relationships loaded into each return invite.
func (h Invite) IndexWithRelations(s *namespace.InviteStore) ([]*namespace.Invite, error) {
	ii, err := s.All()

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := namespace.LoadInviteRelations(h.loaders, ii...); err != nil {
		return ii, errors.Err(err)
	}

	mm := make([]database.Model, 0, len(ii))

	for _, i := range ii {
		mm = append(mm, i.Namespace)
	}

	err = h.Users.Load("id", database.MapKey("user_id", mm), database.Bind("user_id", "id", mm...))
	return ii, errors.Err(err)
}

// StoreModel unmarshals the request's data into an invite, validates it and
// stores it in the database. Upon success this will return the newly created
// invite. This also returns the form for sending an invite.
func (h Invite) StoreModel(r *http.Request) (*namespace.Invite, namespace.InviteForm, error) {
	f := namespace.InviteForm{}

	ctx := r.Context()

	n, ok := namespace.FromContext(ctx)

	if !ok {
		return nil, f, errors.New("no namespace in request context")
	}

	invites := namespace.NewInviteStore(h.DB, n)

	f.Collaborators = namespace.NewCollaboratorStore(h.DB, n)
	f.Invites = invites
	f.Users = h.Users
	f.Owner = mux.Vars(r)["username"]

	if err := webutil.UnmarshalAndValidate(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	i, err := invites.Create(f.Inviter.ID, f.Invitee.ID)

	if err != nil {
		return nil, f, errors.Err(err)
	}

	if h.SMTP.Client != nil {
		m := mail.Mail{
			From:    h.SMTP.From,
			To:      []string{f.Invitee.Email},
			Subject: fmt.Sprintf("Djinn CI - %s invited you to %s", f.Inviter.Username, n.Path),
			Body:    fmt.Sprintf(inviteMail, f.Inviter.Username, n.Path, webutil.BaseAddress(r)),
		}

		return i, f, errors.Err(m.Send(h.SMTP.Client))
	}
	return i, f, nil
}

// Accept accepts an invite in the given request, and adds the user the invite
// was sent to as a collaborator to the invite's namespace. Upon success this
// will return the namespace the user was invited to, the user who sent the
// invite, and the user who received the invite.
func (h Invite) Accept(r *http.Request) (*namespace.Namespace, *user.User, *user.User, error) {
	ctx := r.Context()

	u, ok := user.FromContext(ctx)

	if !ok {
		return nil, nil, nil, errors.New("no user in request context")
	}

	i, ok := namespace.InviteFromContext(r.Context())

	if !ok {
		return nil, nil, nil, errors.New("no invite in request context")
	}

	if i.InviteeID != u.ID {
		return nil, nil, nil, database.ErrNotFound
	}

	n, inviter, invitee, err := namespace.NewInviteStore(h.DB).Accept(i.ID)

	if err != nil {
		return nil, nil, nil, errors.Err(err)
	}

	h.Queue.Enqueue(func() error {
		v := map[string]interface{}{
			"namespace": n.JSON(env.DJINN_API_SERVER),
			"user":      invitee.JSON(env.DJINN_API_SERVER),
		}

		return namespace.NewWebhookStore(h.DB, n).Deliver("collaborator_joined", v)
	})

	return n, inviter, invitee, nil
}

// DeleteModel deletes the namespace invite in the given request context from
// the database.
func (h Invite) DeleteModel(r *http.Request) error {
	ctx := r.Context()

	u, ok := user.FromContext(ctx)

	if !ok {
		return errors.New("no user in request context")
	}

	i, ok := namespace.InviteFromContext(ctx)

	if !ok {
		return errors.New("no invite in request context")
	}

	if i.IsZero() || (u.ID != i.InviteeID && u.ID != i.InviterID) {
		return database.ErrNotFound
	}
	return errors.Err(namespace.NewInviteStore(h.DB).Delete(i))
}
