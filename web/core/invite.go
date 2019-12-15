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
)

type Invite struct {
	web.Handler
}

func (h Invite) Store(w http.ResponseWriter, r *http.Request) (*model.Invite, error) {
	u := h.User(r)

	vars := mux.Vars(r)

	owner, err := h.Users.Get(query.Where("username", "=", vars["username"]))

	if err != nil {
		return &model.Invite{}, errors.Err(err)
	}

	if u.ID != owner.ID {
		return &model.Invite{}, ErrAccessDenied
	}

	n, err := owner.NamespaceStore().Get(query.Where("path", "=", strings.TrimSuffix(vars["namespace"], "/")))

	if err != nil {
		return &model.Invite{}, ErrNotFound
	}

	invites := n.InviteStore()

	f := &form.Invite{
		Collaborators: n.CollaboratorStore(),
		Invites:       invites,
		Users:         h.Users,
		Inviter:       owner,
	}

	if err := h.ValidateForm(f, w, r); err != nil {
		if _, ok := err.(form.Errors); ok {
			return &model.Invite{}, ErrValidationFailed
		}

		return &model.Invite{}, errors.Err(err)
	}

	i := invites.New()
	i.InviteeID = f.Invitee.ID
	i.InviterID = f.Inviter.ID

	err = invites.Create(i)

	return i, errors.Err(err)
}
