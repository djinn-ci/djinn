package web

import (
	"context"
	"net/http"

	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/key"
	"github.com/andrewpillar/djinn/key/handler"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/server"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

type Router struct {
	key handler.Key

	Block      *crypto.Block
	Middleware web.Middleware
}

var _ server.Router = (*Router)(nil)

func Gate(db *sqlx.DB) web.Gate {
	keys := key.NewStore(db)

	return func(u *user.User, r *http.Request) (*http.Request, bool, error) {
		var (
			k   *key.Key
			err error
		)

		ok, err := web.CanAccessResource(db, "key", r, func(id int64) (database.Model, error) {
			k, err = keys.Get(query.Where("id", "=", id))
			return k, errors.Err(err)
		})

		r = r.WithContext(context.WithValue(r.Context(), "key", k))
		return r, ok, errors.Err(err)
	}
}

func (r *Router) Init(h web.Handler) {
	loaders := database.NewLoaders()
	loaders.Put("namespace", namespace.NewStore(h.DB))

	r.key = handler.Key{
		Handler: h,
		Loaders: loaders,
		Block:   r.Block,
		Keys:    key.NewStore(h.DB),
	}
}

func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	key := handler.UI{
		Key: r.key,
	}

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/keys", key.Index).Methods("GET")
	auth.HandleFunc("/keys/create", key.Create).Methods("GET")
	auth.HandleFunc("/keys", key.Store).Methods("POST")
	auth.Use(r.Middleware.AuthPerms("key:read", "key:write"), csrf)

	sr := mux.PathPrefix("/keys").Subrouter()
	sr.HandleFunc("/{key:[0-9]+}/edit", key.Edit).Methods("GET")
	sr.HandleFunc("/{key:[0-9]+}", key.Update).Methods("PATCH")
	sr.HandleFunc("/{key:[0-9]+}", key.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...), csrf)
}

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
	sr.Use(r.Middleware.Gate(gates...))
}
