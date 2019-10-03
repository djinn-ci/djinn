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
	if _, err := h.Core.Store(w, r); err != nil {
		cause := errors.Err(err)

		log.Error.Println(errors.Err(err))
		h.Core.FlashAlert(w, r, template.Danger("Failed to tag build: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h Tag) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.Core.Destroy(r); err != nil {
		cause := errors.Err(err)

		log.Error.Println(errors.Err(err))
		h.Core.FlashAlert(w, r, template.Danger("Failed to tag build: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.Core.FlashAlert(w, r, template.Success("Tag has been deleted"))

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
