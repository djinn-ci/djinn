package web

import (
	"context"
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/key"
	"github.com/andrewpillar/thrall/key/handler"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/server"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

type Router struct {
	key handler.Key

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

		ok, err := web.CanAccessResource(db, "key", r, func(id int64) (model.Model, error) {
			k, err = keys.Get(query.Where("id", "=", id))
			return k, errors.Err(err)
		})

		r = r.WithContext(context.WithValue(r.Context(), "key", k))
		return r, ok, errors.Err(err)
	}
}

func (r *Router) Init(h web.Handler) {
	namespaces := namespace.NewStore(h.DB)

	loaders := model.NewLoaders()
	loaders.Put("namespace", namespaces)

	r.key = handler.Key{
		Handler:    h,
		Loaders:    loaders,
		Namespaces: namespaces,
		Keys:       key.NewStore(h.DB),
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
	auth.Use(r.Middleware.Auth, csrf)

	sr := mux.PathPrefix("/keys").Subrouter()
	sr.HandleFunc("/{key:[0-9]+}/edit", key.Edit).Methods("GET")
	sr.HandleFunc("/{key:[0-9]+}", key.Update).Methods("PATCH")
	sr.HandleFunc("/{key:[0-9]+}", key.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...), csrf)
}

func (r *Router) RegisterAPI(prefix string, mux *mux.Router, gates ...web.Gate) {

}
