package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

type Invite struct {
	web.Handler

	Loaders model.Loaders
}

func (h Invite) IndexWithRelations(r *http.Request) ([]*namespace.Invite, error) {
	u := h.User(r)

	ii, err := namespace.NewInviteStore(h.DB, u).All(query.Where("invitee_id", "=", u.ID))

	if err != nil {
		return ii, errors.Err(err)
	}

	if err := namespace.LoadInviteRelations(h.Loaders, ii...); err != nil {
		return ii, errors.Err(err)
	}

	nn := make([]model.Model, 0, len(ii))

	for _, i := range ii {
		nn = append(nn, i)
	}

	err = h.Users.Load("id", model.MapKey("user_id", nn), model.Bind("user_id", "id", nn...))
	return ii, errors.Err(err)
}

func (h Invite) StoreModel(r *http.Request, sess *sessions.Session) (*namespace.Invite, error) {
	u := h.User(r)
	vars := mux.Vars(r)

	owner, err := h.Users.Get(query.Where("username", "=", vars["username"]))

	if err != nil {
		return &namespace.Invite{}, errors.Err(err)
	}

	if owner.IsZero() {
		return &namespace.Invite{}, model.ErrNotFound
	}

	if u.ID != owner.ID {
		return &namespace.Invite{}, namespace.ErrPermission
	}

	path := strings.TrimSuffix(vars["namespace"], "/")

	n, err := namespace.NewStore(h.DB, owner).Get(query.Where("path", "=", path))

	if err != nil {
		return &namespace.Invite{}, namespace.ErrPermission
	}

	if n.IsZero() {
		return &namespace.Invite{}, model.ErrNotFound
	}

	invites := namespace.NewInviteStore(h.DB, n)

	f := &namespace.InviteForm{
		Collaborators: namespace.NewCollaboratorStore(h.DB, n),
		Invites:       invites,
		Users:         h.Users,
		Inviter:       owner,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		return &namespace.Invite{}, errors.Err(err)
	}

	i := invites.New()
	i.InviteeID = f.Invitee.ID
	i.InviterID = f.Inviter.ID

	err = invites.Create(i)
	return i, errors.Err(err)
}

func (h Invite) Delete(r *http.Request) error {
	id, err := strconv.ParseInt(mux.Vars(r)["invite"], 10, 64)

	if err != nil {
		return model.ErrNotFound
	}

	invites := namespace.NewInviteStore(h.DB)

	i, err := invites.Get(query.Where("id", "=", id))

	if err != nil {
		return errors.Err(err)
	}

	u := h.User(r)

	if i.IsZero() || u.ID != i.InviteeID {
		return model.ErrNotFound
	}
	return errors.Err(invites.Delete(i))
}
