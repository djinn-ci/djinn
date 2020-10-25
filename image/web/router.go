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

// Router is what registers the UI and API routes for managing driver images. It
// implements the server.Router interface.
type Router struct {
	image handler.Image

	// Middleware is the middleware that is applied to any routes registered
	// from this router.
	Middleware web.Middleware

	// Hasher is the hashing mechanism to use when generating hashes for
	// images.
	Hasher *crypto.Hasher

	// BlockStore is the block store implementation to use for storing images
	// that are uploaded.
	BlockStore block.Store

	// Limit is the maximum limit applied to images uploaded.
	Limit int64
}

var _ server.Router = (*Router)(nil)

// Gate returns a web.Gate that checks if the current authenticated User has
// the access permissions to the current Image. If the current user can access
// the current image then it is set in the request's context.
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

// Init intialises the primary handler.Image for handling the primary logic
// of Cron creation and management. This will setup the database.Loader for
// relationship loading, and the related database stores. The exported
// properties on the Router itself are passed through to the underlying
// handler.Image.
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

// RegisterUI registers the UI routes for working with images. There are two
// types of route groups, simple auth routes, and individual image routes.
// These routes respond with a "text/html" Content-Type.
//
// simple auth routes - These routes are registered under the "/images" prefix
// of the given router. The Auth middleware is applied to all registered routes.
// CSRF protection is applied to all the registered routes.
//
// individual image routes - These routes are registered under the
// "/images/{image:[0-9]}" prefix of the given router. Each given gate is applied
// to the registered routes, along with the given CSRF protection.
func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	image := handler.UI{
		Image: r.image,
	}

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/images", image.Index).Methods("GET")
	auth.HandleFunc("/images/create", image.Create).Methods("GET")
	auth.HandleFunc("/images", image.Store).Methods("POST")
	auth.Use(r.Middleware.Auth, r.Middleware.Gate(gates...), csrf)

	sr := mux.PathPrefix("/images").Subrouter()
	sr.HandleFunc("/{image:[0-9]+}/download/{name}", image.Show).Methods("GET")
	sr.HandleFunc("/{image:[0-9]+}", image.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...), csrf)
}

// RegisterAPI registers the API routes for working with images. The given
// prefix string is used to specify where the API is being served under. This
// applies all of the given gates to all routes registered. These routes
// response with a "application/json" Content-Type.
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
