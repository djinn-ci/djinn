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

type Router struct {
	variable handler.Variable

	Middleware web.Middleware
}

var _ server.Router = (*Router)(nil)

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
