package ui

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/web/core"
)

type Invite struct {
	Core core.Invite
}

func (h Invite) Store(w http.ResponseWriter, r *http.Request) {
	_, err := h.Core.Store(w, r)

	if err != nil {
		cause := errors.Cause(err)

		if cause == core.ErrValidationFailed {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		if cause == core.ErrNotFound || cause == core.ErrAccessDenied {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		log.Error.Println(errors.Err(err))
		h.Core.FlashAlert(w, r, template.Danger("Failed to invite user: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.Core.FlashAlert(w, r, template.Success("Invite sent"))
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h Invite) Destroy(w http.ResponseWriter, r *http.Request) {

}
