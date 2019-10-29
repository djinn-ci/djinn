package ui

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/mux"
)

type Oauth struct {
	web.Handler

	Providers map[string]oauth2.Provider
}

func (h Oauth) Auth(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["provider"]

	provider, ok := h.Providers[name]

	if !ok {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	u := h.User(r)

	if err := provider.Auth(r.Context(), r.URL.Query().Get("code"), u.ProviderStore()); err != nil {
		log.Error.Println(errors.Err(err))

		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	h.FlashAlert(w, r, template.Success("Successfully connected to " + name))

	http.Redirect(w, r, "/", http.StatusSeeOther)
}