package web

import (
	"context"
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/object"
	"github.com/andrewpillar/thrall/object/handler"
	"github.com/andrewpillar/thrall/server"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

type Router struct {
	object handler.Object

	Middleware web.Middleware
	FileStore  filestore.FileStore
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

		ok, err := web.Resource(db, "object", r, func(id int64) (model.Model, error) {
			o, err := objects.Get(query.Where("id", "=", id))
			return o, errors.Err(err)
		})

		r = r.WithContext(context.WithValue(r.Context(), "object", o))
		return r, ok, errors.Err(err)
	}
}

func (r *Router) Init(h web.Handler) {
	loaders := model.NewLoaders()
	loaders.Put("namespace", namespace.NewStore(h.DB))

	r.object = handler.Object{
		Handler:    h,
		Loaders:    loaders,
		Objects:    object.NewStore(h.DB),
		FileStore:  r.FileStore,
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
	auth.Use(r.Middleware.Auth, csrf)

	sr := mux.PathPrefix("/objects").Subrouter()
	sr.HandleFunc("/{object:[0-9]+}", object.Show).Methods("GET")
	sr.HandleFunc("/{object:[0-9]+}/download/{name}", object.Show).Methods("GET")
	sr.HandleFunc("/{object:[0-9]+}", object.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...), csrf)
}

func (r *Router) RegisterAPI(mux *mux.Router, gates ...web.Gate) {

}
