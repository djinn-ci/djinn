package web

import (
	"context"
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/image"
	"github.com/andrewpillar/thrall/image/handler"
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
	image handler.Image

	Middleware web.Middleware
	Store      model.Store
	FileStore  filestore.FileStore
	Limit      int64
}

var _ server.Router = (*Router)(nil)

func Gate(db *sqlx.DB) web.Gate {
	images := image.NewStore(db)

	return func(u *user.User, r *http.Request) (*http.Request, bool, error) {
		var (
			i   *image.Image
			err error
		)

		ok, err := web.Resource(db, "image", r, func(id int64) (model.Model, error) {
			i, err = images.Get(query.Where("id", "=", id))
			return i, errors.Err(err)
		})

		if err != nil {
			return r, false, errors.Err(err)
		}

		r = r.WithContext(context.WithValue(r.Context(), "image", i))
		return r, ok, nil
	}
}

func (r *Router) Init(h web.Handler) {
	loaders := model.NewLoaders()
	loaders.Put("namespace", namespace.NewStore(h.DB))

	r.image = handler.Image{
		Handler:    h,
		Loaders:    loaders,
		Images:     image.NewStore(h.DB),
		FileStore:  r.FileStore,
		Limit:      r.Limit,
	}
}

func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	image := handler.UI{
		Image: r.image,
	}

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/images", image.Index).Methods("GET")
	auth.HandleFunc("/images/create", image.Create).Methods("GET")
	auth.HandleFunc("/images", image.Store).Methods("POST")
	auth.Use(r.Middleware.Auth, csrf)

	sr := mux.PathPrefix("/images").Subrouter()
	sr.HandleFunc("/{image:[0-9]+}/download/{name}", image.Show).Methods("GET")
	sr.HandleFunc("/{image:[0-9]+}", image.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...), csrf)
}

func (r *Router) RegisterAPI(mux *mux.Router, gates ...web.Gate) {
}
