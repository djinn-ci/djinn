package router

import (
	"context"
	"net/http"
	"strconv"

	"github.com/andrewpillar/djinn/config"
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

// Router is what registers the routes to handling the management of OAuth
// tokens, and applications. It implements the server.Router interface.
type Router struct {
	middleware web.Middleware
	oauth2     handler.Oauth2
	app        handler.App
	token      handler.Token
	connection handler.Connection

	//	// Block is the block cipher to use for encrypting client secrets.
	//	Block *crypto.Block
	//
	//	// Middleware is the middleware that is applied to any routes registered
	//	// from this router.
	//	Middleware web.Middleware
}

var _ server.Router = (*Router)(nil)

// Gate returns a web.Gate that checks if the current authenticated user has
// the access permissions to the current token. If the current user can access
// the current token, then it is set in the request's context.
func Gate(db *sqlx.DB) web.Gate {
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

func New(cfg config.Server, h web.Handler, mw web.Middleware) *Router {
	return &Router{
		oauth2:     handler.New(h, cfg.BlockCipher()),
		app:        handler.NewApp(h, cfg.BlockCipher()),
		connection: handler.NewConnection(h),
		token:      handler.NewToken(h),
	}
}

// Init initialises the handlers for the OAuth server, managing OAuth apps and
// tokens. The exported properties on the Router itself are passed through to
// the underlying handlers.
//func (r *Router) Init(h web.Handler) {
//	apps := oauth2.NewAppStore(h.DB)
//	tokens := oauth2.NewTokenStore(h.DB)
//
//	r.oauth2 = handler.Oauth2{
//		Handler: h,
//		Apps:    oauth2.NewAppStoreWithBlock(h.DB, r.Block),
//	}
//	r.app = handler.App{
//		Handler: h,
//		Block:   r.Block,
//		Apps:    apps,
//	}
//	r.connection = handler.Connection{
//		Handler: h,
//		Apps:    apps,
//		Tokens:  tokens,
//	}
//	r.token = handler.Token{
//		Handler: h,
//		Tokens:  tokens,
//	}
//}

// RegisterUI registers the UI routes for handling the OAuth web flow, managing
// OAuth apps, and personal access tokens. No CSRF protection is applied to the
// OAuth web flow routes.
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
	auth.HandleFunc("/connections", r.connection.Index).Methods("GET")
	auth.HandleFunc("/connections/{id}", r.connection.Show).Methods("GET")
	auth.HandleFunc("/connections/{id}", r.connection.Destroy).Methods("DELETE")
	auth.Use(r.middleware.Auth, csrf)

	tok := mux.PathPrefix("/settings/tokens").Subrouter()
	tok.HandleFunc("", r.token.Index).Methods("GET")
	tok.HandleFunc("/create", r.token.Create).Methods("GET")
	tok.HandleFunc("", r.token.Store).Methods("POST")
	tok.HandleFunc("/revoke", r.token.Destroy).Methods("DELETE")
	tok.HandleFunc("/{token}", r.token.Edit).Methods("GET")
	tok.HandleFunc("/{token}", r.token.Update).Methods("PATCH")
	tok.HandleFunc("/{token}/regenerate", r.token.Update).Methods("PATCH")
	tok.HandleFunc("/{token}", r.token.Destroy).Methods("DELETE")
	tok.Use(r.middleware.Gate(gates...), csrf)
}

// RegisterAPI is a stub method to implement the server.Router interface.
func (r *Router) RegisterAPI(_ string, _ *mux.Router, _ ...web.Gate) {}
