package web

import (
	"context"
	"net/http"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/server"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/variable"
	"github.com/andrewpillar/djinn/variable/handler"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

// Router is what registers the UI and API routes for managing variables. It
// implements the server.Router interface.
type Router struct {
	variable handler.Variable

	// Middleware is the middleware that is applied to any routes registered
	// from this router.
	Middleware web.Middleware
}

var _ server.Router = (*Router)(nil)

// Gate returns a web.Gate that checks if the current authenticated User has
// the access permissions to the current Variable. If the current user can
// access the current variable then it is set in the request's context.
func Gate(db *sqlx.DB) web.Gate {
	variables := variable.NewStore(db)

	return func(u *user.User, r *http.Request) (*http.Request, bool, error) {
		var (
			v   *variable.Variable
			err error
		)

		ok, err := web.CanAccessResource(db, "variable", r, func(id int64) (database.Model, error) {
			v, err = variables.Get(query.Where("id", "=", id))
			return v, errors.Err(err)
		})

		r = r.WithContext(context.WithValue(r.Context(), "variable", v))
		return r, ok, errors.Err(err)
	}
}

// Init intialises the primary handler.Variable for handling the primary logic
// of Variable creation and management. This will setup the database.Loader for
// relationship loading, and the related database stores. The exported
// properties on the Router itself are passed through to the underlying
// handler.Variable.
func (r *Router) Init(h web.Handler) {
	loaders := database.NewLoaders()
	loaders.Put("user", h.Users)
	loaders.Put("namespace", namespace.NewStore(h.DB))

	r.variable = handler.Variable{
		Handler:   h,
		Loaders:   loaders,
		Variables: variable.NewStore(h.DB),
	}
}

// RegisterUI registers the UI routes for working with variables There are two
// types of route groups, simple auth routes, and individual variable routes.
// These routes respond with a "text/html" Content-Type.
//
// simple auth routes - These routes are registered under the "/variables"
// prefix of the given router. The Auth middleware is applied to all registered
// routes. CSRF protection is applied to all the registered routes.
//
// individual variable routes - These routes are registered under the
// "/variables/variable:[0-9]}" prefix of the given router. Each given gate is
// applied to the registered routes, along with the given CSRF protection.
func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	variable := handler.UI{
		Variable: r.variable,
	}

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/variables", variable.Index).Methods("GET")
	auth.HandleFunc("/variables/create", variable.Create).Methods("GET")
	auth.HandleFunc("/variables", variable.Store).Methods("POST")
	auth.Use(r.Middleware.Auth, r.Middleware.Gate(gates...), csrf)

	sr := mux.PathPrefix("/variables").Subrouter()
	sr.HandleFunc("/{variable:[0-9]+}", variable.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...), csrf)
}

// RegisterAPI registers the API routes for working with variables. The given
// prefix string is used to specify where the API is being served under. This
// applies all of the given gates to all routes registered. These routes
// response with a "application/json" Content-Type.
func (r *Router) RegisterAPI(prefix string, mux *mux.Router, gates ...web.Gate) {
	variable := handler.API{
		Variable: r.variable,
		Prefix:   prefix,
	}

	sr := mux.PathPrefix("/variables").Subrouter()
	sr.HandleFunc("", variable.Index).Methods("GET", "HEAD")
	sr.HandleFunc("", variable.Store).Methods("POST")
	sr.HandleFunc("/{variable:[0-9]+}", variable.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...))
}
