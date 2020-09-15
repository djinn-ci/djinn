package web

import (
	"net/http"

	"github.com/andrewpillar/djinn/provider"
	"github.com/andrewpillar/djinn/server"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/user/handler"
	"github.com/andrewpillar/djinn/web"

	"github.com/gorilla/mux"
)

type Router struct {
	user handler.User

	Registry   *provider.Registry
	Middleware web.Middleware
}

var _ server.Router = (*Router)(nil)

// Init initializes the current router's underlying handler.User with the
// Router's map of oauth2.Provider interfaces.
func (r *Router) Init(h web.Handler) {
	r.user = handler.User{
		Handler:  h,
		Registry: r.Registry,
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
	guest.Use(r.Middleware.Guest, csrf)

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/settings", r.user.Settings).Methods("GET")
	auth.HandleFunc("/settings/verify", r.user.Verify).Methods("GET", "POST")
	auth.HandleFunc("/settings/email", r.user.Email).Methods("PATCH")
	auth.HandleFunc("/settings/password", r.user.Password).Methods("PATCH")
	auth.HandleFunc("/settings/delete", r.user.Destroy).Methods("POST")
	auth.HandleFunc("/logout", r.user.Logout).Methods("POST")
	auth.Use(r.Middleware.Auth, csrf)
}

func (r *Router) RegisterAPI(prefix string, mux *mux.Router, gates ...web.Gate) {
	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		u, _ := user.FromContext(r.Context())
		web.JSON(w, u.JSON(web.BaseAddress(r)+"/"+prefix), http.StatusOK)
	})
	auth.Use(r.Middleware.Auth)
}
