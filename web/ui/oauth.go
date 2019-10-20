package ui

import (
	"net/http"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/mux"

	"golang.org/x/oauth2"
)

type Oauth struct {
	web.Handler

	Configs map[string]*oauth2.Config
}

func (h Oauth) Auth(w http.ResponseWriter, r *http.Request) {
	provider := mux.Vars(r)["provider"]

	cfg := h.Configs[provider]

	tok, err := cfg.Exchange(r.Context(), r.URL.Query().Get("code"))

	if err != nil {
		log.Error.Println(errors.Err(err))

		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	u := h.User(r)

	providers := u.ProviderStore()

	access, err := crypto.Encrypt([]byte(tok.AccessToken))

	if err != nil {
		log.Error.Println(errors.Err(err))

		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	refresh, err := crypto.Encrypt([]byte(tok.RefreshToken))

	if err != nil {
		log.Error.Println(errors.Err(err))

		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p, err := providers.FindByName(provider)

	if err != nil {
		log.Error.Println(errors.Err(err))

		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	create := p.IsZero()

	p.Name = provider
	p.AccessToken = access
	p.RefreshToken = refresh
	p.ExpiresAt = tok.Expiry

	if create {
		err = providers.Create(p)
	} else {
		err = providers.Update(p)
	}

	if err != nil {
		log.Error.Println(errors.Err(err))

		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	h.FlashAlert(w, r, template.Success("Successfully connected to GitHub"))

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
