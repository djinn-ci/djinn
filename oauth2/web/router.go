package web

import (
	"context"
	"net/http"
	"strconv"

	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/oauth2"
	"github.com/andrewpillar/djinn/oauth2/handler"
	"github.com/andrewpillar/djinn/server"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

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
}

var _ server.Router = (*Router)(nil)

func tokenGate(db *sqlx.DB) web.Gate {
	return func(u *user.User, r *http.Request) (*http.Request, bool, error) {
		base := web.BasePath(r.URL.Path)

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
		Handler: h,
		Apps:    oauth2.NewAppStoreWithBlock(h.DB, r.Block),
	}
	r.app = handler.App{
		Handler: h,
		Block:   r.Block,
		Apps:    oauth2.NewAppStore(h.DB),
	}
	r.token = handler.Token{
		Handler: h,
		Tokens:  oauth2.NewTokenStore(h.DB),
	}
}

func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	mux.HandleFunc("/login/oauth/authorize", r.oauth2.Auth).Methods("GET", "POST")
	mux.HandleFunc("/login/oauth/token", r.oauth2.Token).Methods("POST")
	mux.HandleFunc("/logout/oauth/revoke", r.oauth2.Revoke).Methods("POST")

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
}

func (r *Router) RegisterAPI(_ string, _ *mux.Router, _ ...web.Gate) {}
