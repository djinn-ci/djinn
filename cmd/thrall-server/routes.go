package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

func registerRoutes(dir string) *mux.Router {
	r := mux.NewRouter()

	assetsHandler := http.StripPrefix("/assets/", http.FileServer(http.Dir(dir)))

	r.PathPrefix("/assets/").Handler(assetsHandler)

	return r
}
