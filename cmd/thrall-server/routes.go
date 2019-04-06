package main

import (
	"net/http"

	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/mux"
)

func registerWebRoutes(h web.Handler, dir string) *mux.Router {
	r := mux.NewRouter()

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		web.HTMLError(w, "Not found", http.StatusNotFound)
	})

	r.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		web.HTMLError(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	assetsHandler := http.StripPrefix("/assets/", http.FileServer(http.Dir(dir)))

	r.PathPrefix("/assets/").Handler(assetsHandler)

	page := web.NewPage(h)

	r.HandleFunc("/", page.Home)

	auth := web.NewAuth(h)
	mw := web.NewMiddleware(h)

	r.HandleFunc("/register", mw.Guest(auth.Register)).Methods("GET", "POST")
	r.HandleFunc("/login", mw.Guest(auth.Login)).Methods("GET", "POST")
	r.HandleFunc("/logout", mw.Auth(auth.Logout)).Methods("POST")

	namespaceRoutes(r, h, mw)
	buildRoutes(r, h, mw)

	return r
}

func namespaceRoutes(r *mux.Router, h web.Handler, mw web.Middleware) {
	log.Debug.Println("registering namespace routes")

	namespace := web.NewNamespace(h)

	r.HandleFunc("/namespaces", mw.Auth(namespace.Index)).Methods("GET")
	r.HandleFunc("/namespaces/create", mw.Auth(namespace.Create)).Methods("GET")
	r.HandleFunc("/namespaces", mw.Auth(namespace.Store)).Methods("POST")

	r.HandleFunc("/u/{username}/{namespace:[a-zA-Z0-9\\/?\\S]+}/-/edit", mw.Auth(namespace.Edit)).Methods("GET")
	r.HandleFunc("/u/{username}/{namespace:[a-zA-Z0-9\\/?\\S]+}/-/namespaces", namespace.Show).Methods("GET")
	r.HandleFunc("/u/{username}/{namespace:[a-zA-Z0-9\\/?\\S]+}", namespace.Show).Methods("GET")
	r.HandleFunc("/u/{username}/{namespace:[a-zA-Z0-9\\/?\\S]+}", mw.Auth(namespace.Update)).Methods("PATCH")
	r.HandleFunc("/u/{username}/{namespace:[a-zA-Z0-9\\/?\\S]+}", mw.Auth(namespace.Destroy)).Methods("DELETE")
}

func buildRoutes(r *mux.Router, h web.Handler, mw web.Middleware) {
	log.Debug.Println("registering build routes")

	build := web.NewBuild(h)

	r.HandleFunc("/builds/create", mw.Auth(build.Create)).Methods("GET")
	r.HandleFunc("/builds", mw.Auth(build.Store)).Methods("POST")
	r.HandleFunc("/builds/{build}", mw.Auth(build.Show)).Methods("GET")
	r.HandleFunc("/builds/{build}/manifest", mw.Auth(build.Show)).Methods("GET")
	r.HandleFunc("/builds/{build}/manifest/raw", mw.Auth(build.Show)).Methods("GET")
}
