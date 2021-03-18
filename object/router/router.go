package router

import (
	"context"
	"net/http"

	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/object"
	"github.com/andrewpillar/djinn/object/handler"
	"github.com/andrewpillar/djinn/server"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

// Router is what registers the UI and API routes for managing objects. It
// implements the server.Router interface.
type Router struct {
	middleware web.Middleware
	object     handler.Object
}

var _ server.Router = (*Router)(nil)

// Gate returns a web.Gate that checks if the current authenticated User has
// the access permissions to the current Object. If the current user can access
// the current object then it is set in the request's context.
func Gate(db *sqlx.DB) web.Gate {
	objects := object.NewStore(db)

	return func(u *user.User, r *http.Request) (*http.Request, bool, error) {
		var (
			o   *object.Object
			err error
		)

		ok, err := web.CanAccessResource(db, "object", r, func(id int64) (database.Model, error) {
			o, err = objects.Get(query.Where("id", "=", query.Arg(id)))
			return o, errors.Err(err)
		})

		r = r.WithContext(context.WithValue(r.Context(), "object", o))
		return r, ok, errors.Err(err)
	}
}

func New(cfg *config.Server, h web.Handler, mw web.Middleware) *Router {
	objects := cfg.Objects()

	return &Router{
		middleware: mw,
		object:     handler.New(h, cfg.Hasher(), objects.Store, objects.Limit),
	}
}

// RegisterUI registers the UI routes for working with objects. There are two
// types of route groups, simple auth routes, and individual object routes.
// These routes respond with a "text/html" Content-Type.
//
// simple auth routes - These routes are registered under the "/objects" prefix
// of the given router. The Auth middleware is applied to all registered routes.
// CSRF protection is applied to all the registered routes.
//
// individual object routes - These routes are registered under the
// "/objects/{object:[0-9]}" prefix of the given router. Each given gate is
// applied to the registered routes, along with the given CSRF protection.
func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	object := handler.UI{
		Object: r.object,
	}

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/objects", object.Index).Methods("GET")
	auth.HandleFunc("/objects/create", object.Create).Methods("GET")
	auth.HandleFunc("/objects", object.Store).Methods("POST")
	auth.Use(r.middleware.Auth, r.middleware.Gate(gates...), csrf)

	sr := mux.PathPrefix("/objects").Subrouter()
	sr.HandleFunc("/{object:[0-9]+}", object.Show).Methods("GET")
	sr.HandleFunc("/{object:[0-9]+}/download/{name}", object.Show).Methods("GET")
	sr.HandleFunc("/{object:[0-9]+}", object.Destroy).Methods("DELETE")
	sr.Use(r.middleware.Gate(gates...), csrf)
}

// RegisterAPI registers the API routes for working with objects. The given
// prefix string is used to specify where the API is being served under. This
// applies all of the given gates to all routes registered. These routes
// response with a "application/json" Content-Type.
func (r *Router) RegisterAPI(prefix string, mux *mux.Router, gates ...web.Gate) {
	object := handler.API{
		Object: r.object,
		Prefix: prefix,
	}

	sr := mux.PathPrefix("/objects").Subrouter()
	sr.HandleFunc("", object.Index).Methods("GET", "HEAD")
	sr.HandleFunc("", object.Store).Methods("POST")
	sr.HandleFunc("/{object:[0-9]+}", object.Show).Methods("GET")
	sr.HandleFunc("/{object:[0-9]+}/builds", object.Show).Methods("GET")
	sr.HandleFunc("/{object:[0-9]+}", object.Destroy).Methods("DELETE")
	sr.Use(r.middleware.Gate(gates...))
}
