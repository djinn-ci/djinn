package ui

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/web/core"
)

type Invite struct {
	Core core.Invite
}

func (h Invite) Store(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	if _, err := h.Core.Store(r, sess); err != nil {
		cause := errors.Cause(err)

		switch cause {
		case form.ErrValidation:
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		case model.ErrNotFound:
			fallthrough
		case model.ErrPermission:
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to invite user: " + cause.Error()), "alert")
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	sess.AddFlash(template.Success("Invite sent"), "alert")
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h Invite) Destroy(w http.ResponseWriter, r *http.Request) {

}
