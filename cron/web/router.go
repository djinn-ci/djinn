package web

import (
	"context"
	"net/http"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/cron"
	"github.com/andrewpillar/djinn/cron/handler"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/server"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

// Router is what registers the UI and API routes for managing cron jobs. It
// implements the server.Router interface.
type Router struct {
	cron handler.Cron

	// Middleware is the middleware that is applied to any routes registered
	// from this router.
	Middleware web.Middleware
}

var _ server.Router = (*Router)(nil)

// Gate returns a web.Gate that checks if the current authenticated User has
// the access permissions to the current Cron. If the current user can access
// the current cron then it is set in the request's context.
func Gate(db *sqlx.DB) web.Gate {
	crons := cron.NewStore(db)

	return func(u *user.User, r *http.Request) (*http.Request, bool, error) {
		var (
			c   *cron.Cron
			err error
		)

		ok, err := web.CanAccessResource(db, "cron", r, func(id int64) (database.Model, error) {
			c, err = crons.Get(query.Where("id", "=", id))
			return c, errors.Err(err)
		})

		r = r.WithContext(context.WithValue(r.Context(), "cron", c))
		return r, ok, errors.Err(err)
	}
}

// Init intialises the primary handler.Cron for handling the primary logic
// of Cron creation and management. This will setup the database.Loader for
// relationship loading, and the related database stores. The exported
// properties on the Router itself are passed through to the underlying
// handler.Cron.
func (r *Router) Init(h web.Handler) {
	loaders := database.NewLoaders()
	loaders.Put("user", h.Users)
	loaders.Put("namespace", namespace.NewStore(h.DB))
	loaders.Put("build_tag", build.NewTagStore(h.DB))
	loaders.Put("build_trigger", build.NewTriggerStore(h.DB))

	r.cron = handler.Cron{
		Handler: h,
		Loaders: loaders,
		Crons:   cron.NewStore(h.DB),
		Builds:  build.NewStore(h.DB),
	}
}

// RegisterUI registers the UI routes for working with cron jobs. There are two
// types of route groups, simple auth routes, and individual cron job routes.
// These routes respond with a "text/html" Content-Type.
//
// simple auth routes - These routes are registered under the "/cron" prefix of
// the given router. The Auth middleware is applied to all registered routes.
// CSRF protection is applied to all the registered routes.
//
// individual cron routes - These routes are registered under the
// "/cron/{cron:[0-9]}" prefix of the given router. Each given gate is applied
// to the registered routes, along with the given CSRF protection.
func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	cron := handler.UI{
		Cron: r.cron,
	}

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/cron", cron.Index).Methods("GET")
	auth.HandleFunc("/cron/create", cron.Create).Methods("GET")
	auth.HandleFunc("/cron", cron.Store).Methods("POST")
	auth.Use(r.Middleware.Auth, r.Middleware.Gate(gates...), csrf)

	sr := mux.PathPrefix("/cron").Subrouter()
	sr.HandleFunc("/{cron:[0-9]+}", cron.Show).Methods("GET")
	sr.HandleFunc("/{cron:[0-9]+}/edit", cron.Edit).Methods("GET")
	sr.HandleFunc("/{cron:[0-9]+}", cron.Update).Methods("PATCH")
	sr.HandleFunc("/{cron:[0-9]+}", cron.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...), csrf)
}

// RegisterAPI registers the API routes for working with cron jobs. The given
// prefix string is used to specify where the API is being served under. This
// applies all of the given gates to all routes registered. These routes
// response with a "application/json" Content-Type.
func (r *Router) RegisterAPI(prefix string, mux *mux.Router, gates ...web.Gate) {
	cron := handler.API{
		Cron:   r.cron,
		Prefix: prefix,
	}

	sr := mux.PathPrefix("/cron").Subrouter()
	sr.HandleFunc("", cron.Index).Methods("GET")
	sr.HandleFunc("", cron.Store).Methods("POST")
	sr.HandleFunc("/{cron:[0-9]+}", cron.Show).Methods("GET")
	sr.HandleFunc("/{cron:[0-9]+}/builds", cron.Show).Methods("GET")
	sr.HandleFunc("/{cron:[0-9]+}", cron.Update).Methods("PATCH")
	sr.HandleFunc("/{cron:[0-9]+}", cron.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...))
}
