package handler

import (
	"fmt"
	"net/http"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/mail"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/gorilla/mux"
)

type Invite struct {
	web.Handler

	Loaders *database.Loaders
	Invites *namespace.InviteStore
}

var inviteMail = `%s has invited you to be a collaborator in %s. You can accept this invite via
your Invites list,

    %s/invites`

func (h Invite) IndexWithRelations(s *namespace.InviteStore) ([]*namespace.Invite, error) {
	ii, err := s.All()

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := namespace.LoadInviteRelations(h.Loaders, ii...); err != nil {
		return ii, errors.Err(err)
	}

	mm := make([]database.Model, 0, len(ii))

	for _, i := range ii {
		mm = append(mm, i.Namespace)
	}

	err = h.Users.Load("id", database.MapKey("user_id", mm), database.Bind("user_id", "id", mm...))
	return ii, errors.Err(err)
}

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

	if err := form.UnmarshalAndValidate(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	i, err := invites.Create(f.Inviter.ID, f.Invitee.ID)

	if err != nil {
		return nil, f, errors.Err(err)
	}

	m := mail.Mail{
		From:    h.SMTP.From,
		To:      []string{f.Invitee.Email},
		Subject: fmt.Sprintf("Djinn - %s invited you to %s", f.Inviter.Username, n.Path),
		Body:    fmt.Sprintf(inviteMail, f.Inviter.Username, n.Path, web.BaseAddress(r)),
	}

	return i, f, errors.Err(m.Send(h.SMTP.Client))
}

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
	return n, inviter, invitee, errors.Err(err)
}

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
	return errors.Err(h.Invites.Delete(i))
}
