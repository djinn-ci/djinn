package ui

import (
	"net/http"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/template"

	"github.com/gorilla/mux"
)

type Invite struct {
	web.Handler
}

func (h Invite) Store(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to invite user: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	vars := mux.Vars(r)

	owner, err := h.Users.FindByUsername(vars["username"])

	if err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to invite user: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if u.ID != owner.ID {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	n, err := owner.NamespaceStore().FindByPath(strings.TrimSuffix(vars["namespace"], "/"))

	if err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to invite user: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
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
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to invite user: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	i := invites.New()
	i.InviteeID = f.Invitee.ID
	i.InviterID = f.Inviter.ID

	if err := invites.Create(i); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to invite user: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Invite sent to: " + f.Invitee.Username))
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h Invite) Destroy(w http.ResponseWriter, r *http.Request) {

}
