package web

import (
	"context"
	"net/http"

	"github.com/andrewpillar/djinn/block"
	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/object"
	"github.com/andrewpillar/djinn/object/handler"
	"github.com/andrewpillar/djinn/server"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

type Router struct {
	object handler.Object

	Middleware web.Middleware
	Hasher     *crypto.Hasher
	BlockStore block.Store
	Limit      int64
}

var _ server.Router = (*Router)(nil)

func Gate(db *sqlx.DB) web.Gate {
	objects := object.NewStore(db)

	return func(u *user.User, r *http.Request) (*http.Request, bool, error) {
		var (
			o   *object.Object
			err error
		)

		ok, err := web.CanAccessResource(db, "object", r, func(id int64) (database.Model, error) {
			o, err = objects.Get(query.Where("id", "=", id))
			return o, errors.Err(err)
		})

		r = r.WithContext(context.WithValue(r.Context(), "object", o))
		return r, ok, errors.Err(err)
	}
}

func (r *Router) Init(h web.Handler) {
	loaders := database.NewLoaders()
	loaders.Put("user", h.Users)
	loaders.Put("namespace", namespace.NewStore(h.DB))
	loaders.Put("build_tag", build.NewTagStore(h.DB))
	loaders.Put("build_trigger", build.NewTriggerStore(h.DB))

	r.object = handler.Object{
		Handler:    h,
		Loaders:    loaders,
		Objects:    object.NewStoreWithBlockStore(h.DB, r.BlockStore),
		Builds:     build.NewStore(h.DB),
		Hasher:     r.Hasher,
		BlockStore: r.BlockStore,
		Limit:      r.Limit,
	}
}

func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	object := handler.UI{
		Object: r.object,
	}

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/objects", object.Index).Methods("GET")
	auth.HandleFunc("/objects/create", object.Create).Methods("GET")
	auth.HandleFunc("/objects", object.Store).Methods("POST")
	auth.Use(r.Middleware.Auth, r.Middleware.Gate(gates...), csrf)

	sr := mux.PathPrefix("/objects").Subrouter()
	sr.HandleFunc("/{object:[0-9]+}", object.Show).Methods("GET")
	sr.HandleFunc("/{object:[0-9]+}/download/{name}", object.Show).Methods("GET")
	sr.HandleFunc("/{object:[0-9]+}", object.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...), csrf)
}

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
	sr.Use(r.Middleware.Gate(gates...))
}
