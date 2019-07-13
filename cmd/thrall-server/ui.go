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
	"github.com/andrewpillar/thrall/filestore"

	"github.com/jmoiron/sqlx"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/mux"

	"github.com/go-redis/redis"
)

type uiServer struct {
	*server.Server

	db     *sqlx.DB
	client *redis.Client

	limit int64

	objects   filestore.FileStore
	artifacts filestore.FileStore

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
	namespace := ui.NewNamespace(h, &model.NamespaceStore{DB: s.db})

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
	build := ui.NewBuild(h, s.Queues, &model.NamespaceStore{DB: s.db})

	s.router.HandleFunc("/", mw.Auth(build.Index)).Methods("GET")
	s.router.HandleFunc("/builds/create", mw.Auth(build.Create)).Methods("GET")
	s.router.HandleFunc("/builds", mw.Auth(build.Store)).Methods("POST")

	s.router.HandleFunc("/builds/{build}", mw.Auth(build.Show)).Methods("GET")

	s.router.HandleFunc("/builds/{build}/manifest", mw.Auth(build.ShowMeta)).Methods("GET")
	s.router.HandleFunc("/builds/{build}/manifest/raw", mw.Auth(build.ShowMeta)).Methods("GET")
	s.router.HandleFunc("/builds/{build}/output", mw.Auth(build.ShowMeta)).Methods("GET")
	s.router.HandleFunc("/builds/{build}/output/raw", mw.Auth(build.ShowMeta)).Methods("GET")

	s.router.HandleFunc("/builds/{build}/objects", mw.Auth(build.IndexRelation)).Methods("GET")
	s.router.HandleFunc("/builds/{build}/variables", mw.Auth(build.IndexRelation)).Methods("GET")
}

func (s *uiServer) initJob(h web.Handler, mw web.Middleware) {
	job := ui.NewJob(h)

	s.router.HandleFunc("/builds/{build}/jobs/{job}", mw.Auth(job.Show))
	s.router.HandleFunc("/builds/{build}/jobs/{job}/output/raw", mw.Auth(job.Show))
}

func (s *uiServer) initArtifact(h web.Handler, mw web.Middleware) {
	artifact := ui.NewArtifact(h, s.artifacts)

	s.router.HandleFunc("/builds/{build}/artifacts", mw.Auth(artifact.Index))
	s.router.HandleFunc("/builds/{build}/artifacts/{artifact}/download/{name}", mw.Auth(artifact.Show))
}

func (s *uiServer) initTag(h web.Handler, mw web.Middleware) {
	tag := ui.NewTag(h)

	s.router.HandleFunc("/builds/{build}/tags", mw.Auth(tag.Index)).Methods("GET")
	s.router.HandleFunc("/builds/{build}/tags", mw.Auth(tag.Store)).Methods("POST")
	s.router.HandleFunc("/builds/{build}/tags/{tag}", mw.Auth(tag.Destroy)).Methods("DELETE")
}

func (s *uiServer) initObject(h web.Handler, mw web.Middleware) {
	object := ui.NewObject(h, s.objects, s.limit)

	s.router.HandleFunc("/objects", mw.Auth(object.Index)).Methods("GET")
	s.router.HandleFunc("/objects/create", mw.Auth(object.Create)).Methods("GET")
	s.router.HandleFunc("/objects", mw.Auth(object.Store)).Methods("POST")
	s.router.HandleFunc("/objects/{object}", mw.Auth(object.Show)).Methods("GET")
	s.router.HandleFunc("/objects/{object}/download/{name}", mw.Auth(object.Download))
	s.router.HandleFunc("/objects/{object}", mw.Auth(object.Destroy)).Methods("DELETE")
}

func (s *uiServer) init() {
	gob.Register(form.NewErrors())
	gob.Register(make(map[string]string))

	s.router = mux.NewRouter()

	wh := web.New(
		securecookie.New(s.hash, s.key),
		session.New(s.client, s.key),
		model.UserStore{DB: s.db},
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
	s.initArtifact(wh, mw)
	s.initObject(wh, mw)

	s.Server.Init(web.NewLog(web.NewSpoof(s.router)))
}
