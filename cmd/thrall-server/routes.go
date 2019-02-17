package main

import (
	"net/http"

	"github.com/andrewpillar/thrall/handler"

	"github.com/gorilla/mux"
)

func registerRoutes(h handler.Handler, dir string) *mux.Router {
	r := mux.NewRouter()

	assetsHandler := http.StripPrefix("/assets/", http.FileServer(http.Dir(dir)))

	r.PathPrefix("/assets/").Handler(assetsHandler)

	page := handler.NewPage(h)

	r.HandleFunc("/", page.Home)

	auth := handler.NewAuth(h)
	mw := handler.NewMiddleware(h)

	r.HandleFunc("/register", mw.Guest(auth.Register)).Methods("GET", "POST")
	r.HandleFunc("/login", mw.Guest(auth.Login)).Methods("GET", "POST")

	return r
}
