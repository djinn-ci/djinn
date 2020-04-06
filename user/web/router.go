package web

import (
	"net/http"

	"github.com/andrewpillar/thrall/server"
	"github.com/andrewpillar/thrall/user/handler"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/mux"
)

type Router struct {
	user handler.User

	Middleware web.Middleware
}

var _ server.Router = (*Router)(nil)

func (r *Router) Init(h web.Handler) {
	r.user = handler.User{
		Handler: h,
	}
}

func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	guest := mux.PathPrefix("/").Subrouter()
	guest.HandleFunc("/register", r.user.Register).Methods("GET", "POST")
	guest.HandleFunc("/login", r.user.Login).Methods("GET", "POST")
	guest.Use(r.Middleware.Guest, csrf)

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/settings", r.user.Settings).Methods("GET")
	auth.HandleFunc("/settings/invites", r.user.Settings).Methods("GET")
	auth.HandleFunc("/settings/email", r.user.Email).Methods("PATCH")
	auth.HandleFunc("/settings/password", r.user.Password).Methods("PATCH")
	auth.HandleFunc("/logout", r.user.Logout).Methods("POST")
	auth.Use(r.Middleware.Auth, csrf)
}

func (r *Router) RegisterAPI(mux *mux.Router, gates ...web.Gate) {}
