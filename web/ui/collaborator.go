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
	sess, save := h.Core.Session(r)
	defer save(r, w)

	if err := h.Core.Destroy(r); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		sess.AddFlash(template.Danger("Failed to remove collaborator: " + cause.Error()), "alert")
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	sess.AddFlash(template.Success("Collaborator removed: " + mux.Vars(r)["collaborator"]), "alert")
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h Collaborator) Store(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	n, err := h.Core.Store(r)

	if err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		sess.AddFlash(template.Danger("Failed to accept invite: " + cause.Error()), "alert")
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	sess.AddFlash(template.Success("You are now a collaborator in: " + n.Name), "alert")
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
