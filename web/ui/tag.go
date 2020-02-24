package ui

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/web/core"
	"github.com/andrewpillar/thrall/template"
)

type Tag struct {
	Core core.Tag
}

func (h Tag) Store(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	if _, err := h.Core.Store(w, r); err != nil {
		cause := errors.Err(err)

		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to tag build: " + cause.Error()), "alert")
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h Tag) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	if err := h.Core.Destroy(r); err != nil {
		cause := errors.Err(err)

		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to tag build: " + cause.Error()), "alert")
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	sess.AddFlash(template.Success("Tag has been deleted"), "alert")
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
