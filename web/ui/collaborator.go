package ui

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/web/core"

	"github.com/gorilla/mux"
)

type Collaborator struct {
	Core core.Collaborator
}

func (h Collaborator) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.Core.Destroy(r); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.Core.FlashAlert(w, r, template.Danger("Failed to remove collaborator: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	vars := mux.Vars(r)

	h.Core.FlashAlert(w, r, template.Success("Collaborator removed: " + vars["collaborator"]))
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h Collaborator) Store(w http.ResponseWriter, r *http.Request) {
	n, err := h.Core.Store(r)

	if err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.Core.FlashAlert(w, r, template.Danger("Failed to accept invite: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.Core.FlashAlert(w, r, template.Success("You are now a collaborator in: " + n.Name))
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
