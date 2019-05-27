package main

import (
	"encoding/gob"
	"net/http"

	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/web/ui"
	"github.com/andrewpillar/thrall/server"
	"github.com/andrewpillar/thrall/session"

	"github.com/jmoiron/sqlx"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/mux"

	"github.com/go-redis/redis"
)

type uiServer struct {
	*server.Server

	db     *sqlx.DB
	client *redis.Client

	hash []byte
	key  []byte

	assets string
	router *mux.Router
}

func (s *uiServer) initAuth(h web.Handler, mw web.Middleware) {
	auth := ui.NewAuth(h)

	s.router.HandleFunc("/register", mw.Guest(auth.Register)).Methods("GET", "POST")
	s.router.HandleFunc("/login", mw.Guest(auth.Login)).Methods("GET", "POST")
	s.router.HandleFunc("/logout", mw.Auth(auth.Logout)).Methods("POST")
}

func (s *uiServer) initNamespace(h web.Handler, mw web.Middleware) {
	namespace := ui.NewNamespace(h, model.NewNamespaceStore(s.db))

	s.router.HandleFunc("/namespaces", mw.Auth(namespace.Index)).Methods("GET")
	s.router.HandleFunc("/namespaces/create", mw.Auth(namespace.Create)).Methods("GET")
	s.router.HandleFunc("/namespaces", mw.Auth(namespace.Store)).Methods("POST")

	s.router.HandleFunc("/u/{username}/{namespace:[a-zA-Z0-9\\/?\\S]+}/-/edit", mw.Auth(namespace.Edit)).Methods("GET")
	s.router.HandleFunc("/u/{username}/{namespace:[a-zA-Z0-9\\/?\\S]+}/-/namespaces", namespace.Show).Methods("GET")
	s.router.HandleFunc("/u/{username}/{namespace:[a-zA-Z0-9\\/?\\S]+}", namespace.Show).Methods("GET")
	s.router.HandleFunc("/u/{username}/{namespace:[a-zA-Z0-9\\/?\\S]+}", mw.Auth(namespace.Update)).Methods("PATCH")
	s.router.HandleFunc("/u/{username}/{namespace:[a-zA-Z0-9\\/?\\S]+}", mw.Auth(namespace.Destroy)).Methods("DELETE")
}

func (s *uiServer) initBuild(h web.Handler, mw web.Middleware) {
	build := ui.NewBuild(h, s.Queues, model.NewNamespaceStore(s.db))

	s.router.HandleFunc("/", mw.Auth(build.Index)).Methods("GET")
	s.router.HandleFunc("/builds/create", mw.Auth(build.Create)).Methods("GET")
	s.router.HandleFunc("/builds", mw.Auth(build.Store)).Methods("POST")
	s.router.HandleFunc("/builds/{build}", mw.Auth(build.Show)).Methods("GET")
	s.router.HandleFunc("/builds/{build}/manifest", mw.Auth(build.Show)).Methods("GET")
	s.router.HandleFunc("/builds/{build}/manifest/raw", mw.Auth(build.Show)).Methods("GET")
	s.router.HandleFunc("/builds/{build}/output", mw.Auth(build.Show)).Methods("GET")
	s.router.HandleFunc("/builds/{build}/output/raw", mw.Auth(build.Show)).Methods("GET")
}

func (s *uiServer) initJob(h web.Handler, mw web.Middleware) {
	job := ui.NewJob(h)

	s.router.HandleFunc("/builds/{build}/jobs/{job}", mw.Auth(job.Show))
}

func (s *uiServer) init() {
	gob.Register(web.Form(make(map[string]string)))
	gob.Register(form.NewErrors())
	gob.Register(form.Register{})
	gob.Register(form.Login{})
	gob.Register(form.Namespace{})
	gob.Register(form.Build{})

	s.router = mux.NewRouter()

	wh := web.New(
		securecookie.New(s.hash, s.key),
		session.New(s.client, s.key),
		model.NewUserStore(s.db),
	)
	mw := web.NewMiddleware(wh)

	s.router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		web.HTMLError(w, "Not found", http.StatusNotFound)
	})

	s.router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		web.HTMLError(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	assets := http.StripPrefix("/assets/", http.FileServer(http.Dir(s.assets)))

	s.router.PathPrefix("/assets/").Handler(assets)

	s.initAuth(wh, mw)
	s.initNamespace(wh, mw)
	s.initBuild(wh, mw)
	s.initJob(wh, mw)

	s.Server.Init(web.NewLog(web.NewSpoof(s.router)))
}
