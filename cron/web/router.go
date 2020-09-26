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

type Router struct {
	cron handler.Cron

	Middleware web.Middleware
}

var _ server.Router = (*Router)(nil)

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

func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	cron := handler.UI{
		Cron: r.cron,
	}

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/cron", cron.Index).Methods("GET")
	auth.HandleFunc("/cron/create", cron.Create).Methods("GET")
	auth.HandleFunc("/cron", cron.Store).Methods("POST")
	auth.Use(r.Middleware.Gate(gates...), csrf)

	sr := mux.PathPrefix("/cron").Subrouter()
	sr.HandleFunc("/{cron:[0-9]+}", cron.Show).Methods("GET")
	sr.HandleFunc("/{cron:[0-9]+}/edit", cron.Edit).Methods("GET")
	sr.HandleFunc("/{cron:[0-9]+}", cron.Update).Methods("PATCH")
	sr.HandleFunc("/{cron:[0-9]+}", cron.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...), csrf)
}

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
