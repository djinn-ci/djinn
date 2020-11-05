package router

import (
	"net/http"

	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/server"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/user/handler"
	"github.com/andrewpillar/djinn/web"

	"github.com/gorilla/mux"
)

// Router is what registers the UI routes for handling registration,
// authentication, and general management of a user's account.
type Router struct {
	middleware web.Middleware
	user       handler.User
}

var _ server.Router = (*Router)(nil)

func New(cfg config.Server, h web.Handler, mw web.Middleware) *Router {
	return &Router{
		middleware: mw,
		user:       handler.New(h, cfg.Providers()),
	}
}

// RegisterUI registers the routes for User authentication, and account
// management against the given mux.Router. The given http.Handler is used for
// CSRF protection. None of the given gates are applied to any of the
// registered routes.
func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, _ ...web.Gate) {
	guest := mux.PathPrefix("/").Subrouter()
	guest.HandleFunc("/register", r.user.Register).Methods("GET", "POST")
	guest.HandleFunc("/login", r.user.Login).Methods("GET", "POST")
	guest.HandleFunc("/password_reset", r.user.PasswordReset).Methods("GET", "POST")
	guest.HandleFunc("/new_password", r.user.NewPassword).Methods("GET", "POST")
	guest.Use(r.middleware.Guest, csrf)

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/settings", r.user.Settings).Methods("GET")
	auth.HandleFunc("/settings/verify", r.user.Verify).Methods("GET", "POST")
	auth.HandleFunc("/settings/cleanup", r.user.Cleanup).Methods("PATCH")
	auth.HandleFunc("/settings/email", r.user.Email).Methods("PATCH")
	auth.HandleFunc("/settings/password", r.user.Password).Methods("PATCH")
	auth.HandleFunc("/settings/delete", r.user.Destroy).Methods("POST")
	auth.HandleFunc("/logout", r.user.Logout).Methods("POST")
	auth.Use(r.middleware.Auth, csrf)
}

// RegisterAPI registers the only API route for a user, which is "/user". This
// will return the currently authenticated user from the request.
func (r *Router) RegisterAPI(prefix string, mux *mux.Router, gates ...web.Gate) {
	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		u, _ := user.FromContext(r.Context())
		web.JSON(w, u.JSON(web.BaseAddress(r)+"/"+prefix), http.StatusOK)
	})
	auth.Use(r.middleware.Auth)
}
