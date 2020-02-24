package core

import (
	"net/http"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

type Invite struct {
	web.Handler
}

func (h Invite) Store(r *http.Request, sess *sessions.Session) (*model.Invite, error) {
	u := h.User(r)

	vars := mux.Vars(r)

	owner, err := h.Users.Get(query.Where("username", "=", vars["username"]))

	if err != nil {
		return &model.Invite{}, errors.Err(err)
	}

	if u.ID != owner.ID {
		return &model.Invite{}, model.ErrPermission
	}

	n, err := owner.NamespaceStore().Get(query.Where("path", "=", strings.TrimSuffix(vars["namespace"], "/")))

	if err != nil {
		return &model.Invite{}, model.ErrNotFound
	}

	invites := n.InviteStore()

	f := &form.Invite{
		Collaborators: n.CollaboratorStore(),
		Invites:       invites,
		Users:         h.Users,
		Inviter:       owner,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); ok {
			return &model.Invite{}, form.ErrValidation
		}
		return &model.Invite{}, errors.Err(err)
	}

	i := invites.New()
	i.InviteeID = f.Invitee.ID
	i.InviterID = f.Inviter.ID

	err = invites.Create(i)

	return i, errors.Err(err)
}
