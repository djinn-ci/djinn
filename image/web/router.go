package web

import (
	"context"
	"net/http"

	"github.com/andrewpillar/djinn/block"
	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/image"
	"github.com/andrewpillar/djinn/image/handler"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/server"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

type Router struct {
	image handler.Image

	Middleware web.Middleware
	Store      database.Store
	Hasher     *crypto.Hasher
	BlockStore block.Store
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

		ok, err := web.CanAccessResource(db, "image", r, func(id int64) (database.Model, error) {
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
	loaders := database.NewLoaders()
	loaders.Put("user", h.Users)
	loaders.Put("namespace", namespace.NewStore(h.DB))

	r.image = handler.Image{
		Handler:    h,
		Loaders:    loaders,
		Images:     image.NewStoreWithBlockStore(h.DB, r.BlockStore),
		Hasher:     r.Hasher,
		BlockStore: r.BlockStore,
		Limit:      r.Limit,
	}
}

func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	image := handler.UI{
		Image: r.image,
	}

	auth := mux.PathPrefix("/images").Subrouter()
	auth.HandleFunc("", image.Index).Methods("GET")
	auth.HandleFunc("/create", image.Create).Methods("GET")
	auth.HandleFunc("", image.Store).Methods("POST")
	auth.Use(r.Middleware.Auth, r.Middleware.Gate(gates...), csrf)

	sr := mux.PathPrefix("/images").Subrouter()
	sr.HandleFunc("/{image:[0-9]+}/download/{name}", image.Show).Methods("GET")
	sr.HandleFunc("/{image:[0-9]+}", image.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...), csrf)
}

func (r *Router) RegisterAPI(prefix string, mux *mux.Router, gates ...web.Gate) {
	image := handler.API{
		Image:  r.image,
		Prefix: prefix,
	}

	sr := mux.PathPrefix("/images").Subrouter()
	sr.HandleFunc("", image.Index).Methods("GET", "HEAD")
	sr.HandleFunc("", image.Store).Methods("POST")
	sr.HandleFunc("/{image:[0-9]+}", image.Show).Methods("GET")
	sr.HandleFunc("/{image:[0-9]+}", image.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...))
}
