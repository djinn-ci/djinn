package router

import (
	"context"
	"net/http"

	"djinn-ci.com/config"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/key"
	"djinn-ci.com/key/handler"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	"djinn-ci.com/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

// Router is what registers the UI and API routes for managing SSH keys. It
// implements the server.Router interface.
type Router struct {
	middleware web.Middleware
	key        handler.Key
}

var _ server.Router = (*Router)(nil)

// Gate returns a web.Gate that checks if the current authenticated User has
// the access permissions to the current Key. If the current user can access
// the current key then it is set in the request's context.
func Gate(db *sqlx.DB) web.Gate {
	keys := key.NewStore(db)

	return func(u *user.User, r *http.Request) (*http.Request, bool, error) {
		var (
			k   *key.Key
			err error
		)

		ok, err := web.CanAccessResource(db, "key", r, func(id int64) (database.Model, error) {
			k, err = keys.Get(query.Where("id", "=", query.Arg(id)))
			return k, errors.Err(err)
		})

		r = r.WithContext(context.WithValue(r.Context(), "key", k))
		return r, ok, errors.Err(err)
	}
}

func New(cfg *config.Server, h web.Handler, mw web.Middleware) *Router {
	return &Router{
		middleware: mw,
		key:        handler.New(h, cfg.BlockCipher()),
	}
}

// RegisterUI registers the UI routes for working with keys There are two
// types of route groups, simple auth routes, and individual key routes.
// These routes respond with a "text/html" Content-Type.
//
// simple auth routes - These routes are registered under the "/keys" prefix
// of the given router. The Auth middleware is applied to all registered routes.
// CSRF protection is applied to all the registered routes.
//
// individual key routes - These routes are registered under the
// "/keys/{key:[0-9]}" prefix of the given router. Each given gate is applied
// to the registered routes, along with the given CSRF protection.
func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	key := handler.UI{
		Key: r.key,
	}

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/keys", key.Index).Methods("GET")
	auth.HandleFunc("/keys/create", key.Create).Methods("GET")
	auth.HandleFunc("/keys", key.Store).Methods("POST")
	auth.Use(r.middleware.Auth, r.middleware.Gate(gates...), csrf)

	sr := mux.PathPrefix("/keys").Subrouter()
	sr.HandleFunc("/{key:[0-9]+}/edit", key.Edit).Methods("GET")
	sr.HandleFunc("/{key:[0-9]+}", key.Update).Methods("PATCH")
	sr.HandleFunc("/{key:[0-9]+}", key.Destroy).Methods("DELETE")
	sr.Use(r.middleware.Gate(gates...), csrf)
}

// RegisterAPI registers the API routes for working with keys. The given
// prefix string is used to specify where the API is being served under. This
// applies all of the given gates to all routes registered. These routes
// response with a "application/json" Content-Type.
func (r *Router) RegisterAPI(prefix string, mux *mux.Router, gates ...web.Gate) {
	key := handler.API{
		Key:    r.key,
		Prefix: prefix,
	}

	sr := mux.PathPrefix("/keys").Subrouter()
	sr.HandleFunc("", key.Index).Methods("GET", "HEAD")
	sr.HandleFunc("", key.Store).Methods("POST")
	sr.HandleFunc("/{key:[0-9]+}", key.Update).Methods("PATCH")
	sr.HandleFunc("/{key:[0-9]+}", key.Destroy).Methods("DELETE")
	sr.Use(r.middleware.Gate(gates...))
}
