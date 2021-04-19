package router

import (
	"context"
	"net/http"

	"djinn-ci.com/config"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/image"
	"djinn-ci.com/image/handler"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	"djinn-ci.com/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

// Router is what registers the UI and API routes for managing driver images. It
// implements the server.Router interface.
type Router struct {
	middleware web.Middleware
	image      handler.Image
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
			i, err = images.Get(query.Where("id", "=", query.Arg(id)))
			return i, errors.Err(err)
		})

		if err != nil {
			return r, false, errors.Err(err)
		}

		r = r.WithContext(context.WithValue(r.Context(), "image", i))
		return r, ok, nil
	}
}

func New(cfg *config.Server, h web.Handler, mw web.Middleware) *Router {
	images := cfg.Images()

	return &Router{
		middleware: mw,
		image:      handler.New(h, cfg.Hasher(), images.Store, images.Limit),
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
	auth.Use(r.middleware.Auth, r.middleware.Gate(gates...), csrf)

	sr := mux.PathPrefix("/images").Subrouter()
	sr.HandleFunc("/{image:[0-9]+}/download/{name}", image.Show).Methods("GET")
	sr.HandleFunc("/{image:[0-9]+}", image.Destroy).Methods("DELETE")
	sr.Use(r.middleware.Gate(gates...), csrf)
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
	sr.Use(r.middleware.Gate(gates...))
}
