package router

import (
	"context"
	"net/http"

	"djinn-ci.com/config"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	"djinn-ci.com/variable"
	"djinn-ci.com/variable/handler"
	"djinn-ci.com/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

// Router is what registers the UI and API routes for managing variables. It
// implements the server.Router interface.
type Router struct {
	middleware web.Middleware
	variable   handler.Variable
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
			v, err = variables.Get(query.Where("id", "=", query.Arg(id)))
			return v, errors.Err(err)
		})

		r = r.WithContext(context.WithValue(r.Context(), "variable", v))
		return r, ok, errors.Err(err)
	}
}

func New(_ *config.Server, h web.Handler, mw web.Middleware) *Router {
	return &Router{
		middleware: mw,
		variable:   handler.New(h),
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
	auth.Use(r.middleware.Auth, r.middleware.Gate(gates...), csrf, r.middleware.CheckEmail)

	sr := mux.PathPrefix("/variables").Subrouter()
	sr.HandleFunc("/{variable:[0-9]+}", variable.Destroy).Methods("DELETE")
	sr.Use(r.middleware.Gate(gates...), csrf, r.middleware.CheckEmail)
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
	sr.HandleFunc("/{variable:[0-9]+}", variable.Show).Methods("GET")
	sr.HandleFunc("/{variable:[0-9]+}", variable.Destroy).Methods("DELETE")
	sr.Use(r.middleware.Gate(gates...), r.middleware.CheckEmail)
}
