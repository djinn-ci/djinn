package web

import (
	"context"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/oauth2/handler"
	"github.com/andrewpillar/thrall/server"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

type Router struct {
	oauth2 handler.Oauth2
	app    handler.App
	token  handler.Token

	Block      *crypto.Block
	Middleware web.Middleware
	Providers  map[string]oauth2.Provider
}

var _ server.Router = (*Router)(nil)

func tokenGate(db *sqlx.DB) web.Gate {
	return func(u *user.User, r *http.Request) (*http.Request, bool, error) {
		base := filepath.Base(r.URL.Path)

		if base == "tokens" || base == "create" || base == "revoke" {
			return r, !u.IsZero(), nil
		}

		id, _ := strconv.ParseInt(mux.Vars(r)["token"], 10, 64)

		t, err := oauth2.NewTokenStore(db, u).Get(query.Where("id", "=", id))

		if err != nil {
			return r, false, errors.Err(err)
		}

		if t.IsZero() {
			return r, false, nil
		}

		r = r.WithContext(context.WithValue(r.Context(), "token", t))
		return r, t.UserID == u.ID, nil
	}
}

func (r *Router) Init(h web.Handler) {
	r.oauth2 = handler.Oauth2{
		Handler:   h,
		Apps:      oauth2.NewAppStore(h.DB),
		Providers: r.Providers,
	}
	r.app = handler.App{
		Handler: h,
		Block:   r.Block,
	}
	r.token = handler.Token{
		Handler: h,
		Tokens:  oauth2.NewTokenStore(h.DB),
	}
}

func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	mux.HandleFunc("/oauth/authorize", r.oauth2.Auth).Methods("GET", "POST")
	mux.HandleFunc("/oauth/token", r.oauth2.Token).Methods("POST")
	mux.HandleFunc("/oauth/revoke", r.oauth2.Revoke).Methods("POST")

	auth := mux.PathPrefix("/settings").Subrouter()
	auth.HandleFunc("/apps", r.app.Index).Methods("GET")
	auth.HandleFunc("/apps/create", r.app.Create).Methods("GET")
	auth.HandleFunc("/apps", r.app.Store).Methods("POST")
	auth.HandleFunc("/apps/{app}", r.app.Show).Methods("GET")
	auth.HandleFunc("/apps/{app}", r.app.Update).Methods("PATCH")
	auth.HandleFunc("/apps/{app}/revoke", r.app.Update).Methods("PATCH")
	auth.HandleFunc("/apps/{app}/reset", r.app.Update).Methods("PATCH")
	auth.Use(r.Middleware.Auth, csrf)

	tok := mux.PathPrefix("/settings/tokens").Subrouter()
	tok.HandleFunc("", r.token.Index).Methods("GET")
	tok.HandleFunc("/create", r.token.Create).Methods("GET")
	tok.HandleFunc("", r.token.Store).Methods("POST")
	tok.HandleFunc("/revoke", r.token.Destroy).Methods("DELETE")
	tok.HandleFunc("/{token}", r.token.Edit).Methods("GET")
	tok.HandleFunc("/{token}", r.token.Update).Methods("PATCH")
	tok.HandleFunc("/{token}/regenerate", r.token.Update).Methods("PATCH")
	tok.HandleFunc("/{token}", r.token.Destroy).Methods("DELETE")
	tok.Use(r.Middleware.Gate(tokenGate(r.token.DB)), csrf)

	sr := mux.PathPrefix("/oauth").Subrouter()
	sr.HandleFunc("/{provider}", r.oauth2.AuthClient).Methods("GET")
	sr.HandleFunc("/{provider}", r.oauth2.RevokeClient).Methods("DELETE")
	sr.Use(csrf)
}

func (r *Router) RegisterAPI(_ string, _ *mux.Router, _ ...web.Gate) {}
