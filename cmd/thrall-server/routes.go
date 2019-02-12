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

	return r
}
