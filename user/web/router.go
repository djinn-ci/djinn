package web

import (
	"net/http"

	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/server"
	"github.com/andrewpillar/thrall/user/handler"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/mux"
)

type Router struct {
	user handler.User

	Providers  map[string]oauth2.Provider
	Middleware web.Middleware
}

var _ server.Router = (*Router)(nil)

// Init initializes the current router's underlying handler.User with the
// Router's map of oauth2.Provider interfaces.
func (r *Router) Init(h web.Handler) {
	r.user = handler.User{
		Handler:   h,
		Providers: r.Providers,
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
	guest.Use(r.Middleware.Guest, csrf)

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/settings", r.user.Settings).Methods("GET")
	auth.HandleFunc("/settings/email", r.user.Email).Methods("PATCH")
	auth.HandleFunc("/settings/password", r.user.Password).Methods("PATCH")
	auth.HandleFunc("/logout", r.user.Logout).Methods("POST")
	auth.Use(r.Middleware.Auth, csrf)
}

// RegisterAPI is a stub method to statisfy the server.Router interface.
func (r *Router) RegisterAPI(_ *mux.Router, _ ...web.Gate) {}
