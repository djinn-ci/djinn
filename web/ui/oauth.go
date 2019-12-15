package ui

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

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

	if r.URL.Query().Get("state") != string(provider.Secret()) {
		web.Text(w, "Not found", http.StatusNotFound)
		return
	}

	if err := provider.Auth(r.Context(), r.URL.Query().Get("code"), u.ProviderStore()); err != nil {
		log.Error.Println(errors.Err(err))

		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	h.FlashAlert(w, r, template.Success("Successfully connected to " + name))

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h Oauth) Revoke(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["provider"]

	provider, ok := h.Providers[name]

	if !ok {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	u := h.User(r)

	providers := u.ProviderStore()

	p, err := providers.Get(query.Where("name", "=", name))

	if err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to revoke provider: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if err := provider.Revoke(p); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to revoke provider: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	p.Connected = false
	p.AccessToken = nil
	p.RefreshToken = nil

	if err := providers.Update(p); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to revoke provider: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Successfully revoked access to: " + name))
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
