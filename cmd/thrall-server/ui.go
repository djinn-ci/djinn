package main

import (
	"encoding/gob"
	"net/http"

	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/web/ui"
	"github.com/andrewpillar/thrall/server"
	"github.com/andrewpillar/thrall/session"
	"github.com/andrewpillar/thrall/template"

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
	auth := ui.Auth{
		Handler: h,
	}

	s.router.HandleFunc("/register", auth.Register).Methods("GET", "POST")
	s.router.HandleFunc("/login", auth.Login).Methods("GET", "POST")
	s.router.HandleFunc("/logout", auth.Logout).Methods("POST")
}

func (s *uiServer) initNamespace(h web.Handler, mw web.Middleware) {
	namespace := ui.Namespace{
		Handler:    h,
	}

	authRouter := s.router.PathPrefix("/namespaces").Subrouter()
	authRouter.HandleFunc("", namespace.Index).Methods("GET")
	authRouter.HandleFunc("/create", namespace.Create).Methods("GET")
	authRouter.HandleFunc("", namespace.Store).Methods("POST")
	authRouter.Use(mw.Auth)

	r := s.router.PathPrefix("/u/{username}").Subrouter()
	r.HandleFunc("/{namespace:[a-zA-Z0-9\\/?\\S]+}/-/edit", namespace.Edit).Methods("GET")
	r.HandleFunc("/{namespace:[a-zA-Z0-9\\/?\\S]+}/-/namespaces", namespace.Show).Methods("GET")
	r.HandleFunc("/{namespace:[a-zA-Z0-9\\/?\\S]+}", namespace.Show).Methods("GET")
	r.HandleFunc("/{namespace:[a-zA-Z0-9\\/?\\S]+}", namespace.Update).Methods("PATCH")
	r.HandleFunc("/{namespace:[a-zA-Z0-9\\/?\\S]+}", namespace.Destroy).Methods("DELETE")
	r.Use(mw.AuthNamespace)
}

func (s *uiServer) initBuild(h web.Handler, mw web.Middleware) {
	builds := model.BuildStore{
		DB: s.db,
	}

	build := ui.Build{
		Handler: h,
		Builds:  builds,
		Queues:  s.Queues,
	}

	object := ui.BuildObject{
		Handler: h,
		Builds:  builds,
	}

	variable := ui.BuildVariable{
		Handler: h,
		Builds:  builds,
	}

	s.router.HandleFunc("/", build.Index).Methods("GET")

	r := s.router.PathPrefix("/builds").Subrouter()
	r.HandleFunc("/create", build.Create).Methods("GET")
	r.HandleFunc("", build.Store).Methods("POST")
	r.HandleFunc("/{build:[0-9]+}", build.Show).Methods("GET")
	r.HandleFunc("/{build:[0-9]+}/manifest", build.Show).Methods("GET")
	r.HandleFunc("/{build:[0-9]+}/manifest/raw", build.Show).Methods("GET")
	r.HandleFunc("/{build:[0-9]+}/output", build.Show).Methods("GET")
	r.HandleFunc("/{build:[0-9]+}/output/raw", build.Show).Methods("GET")
	r.HandleFunc("/{build:[0-9]+}/objects", object.Index).Methods("GET")
	r.HandleFunc("/{build:[0-9]+}/variables", variable.Index).Methods("GET")
	r.Use(mw.AuthBuild)
}

func (s *uiServer) initJob(h web.Handler, mw web.Middleware) {
	job := ui.Job{
		Handler: h,
	}

	r := s.router.PathPrefix("/builds/{build:[0-9]+}").Subrouter()
	r.HandleFunc("/jobs/{job:[0-9]+}", job.Show)
	r.HandleFunc("/jobs/{job:[0-9]+}/output/raw", job.Show)
	r.Use(mw.AuthBuild)
}

func (s *uiServer) initArtifact(h web.Handler, mw web.Middleware) {
	artifacts := model.ArtifactStore{
		DB: s.db,
	}

	artifact := ui.Artifact{
		Handler:   h,
		Artifacts: artifacts,
		FileStore: s.artifacts,
	}

	r := s.router.PathPrefix("/builds/{build:[0-9]+}").Subrouter()
	r.HandleFunc("/artifacts", artifact.Index)
	r.HandleFunc("/artifacts/{artifact:[0-9]+}/download/{name}", artifact.Show)
	r.Use(mw.AuthBuild)
}

func (s *uiServer) initTag(h web.Handler, mw web.Middleware) {
	tag := ui.Tag{
		Handler: h,
	}

	r := s.router.PathPrefix("/builds/{build:[0-9]+}").Subrouter()
	r.HandleFunc("/tags", tag.Index).Methods("GET")
	r.HandleFunc("/tags", tag.Store).Methods("POST")
	r.HandleFunc("/tags/{tag:[0-9]+}", tag.Destroy).Methods("DELETE")
	r.Use(mw.AuthBuild)
}

func (s *uiServer) initObject(h web.Handler, mw web.Middleware) {
	object := ui.Object{
		Handler:   h,
		FileStore: s.objects,
		Limit:     s.limit,
	}

	r := s.router.PathPrefix("/objects").Subrouter()
	r.HandleFunc("", object.Index).Methods("GET")
	r.HandleFunc("/create", object.Create).Methods("GET")
	r.HandleFunc("", object.Store).Methods("POST")
	r.HandleFunc("/{object:[0-9]+}", object.Show).Methods("GET")
	r.HandleFunc("/{object:[0-9]+}/download/{name}", object.Download)
	r.HandleFunc("/{object:[0-9]+}", object.Destroy).Methods("DELETE")
	r.Use(mw.AuthObject)
}

func (s *uiServer) initVariable(h web.Handler, mw web.Middleware) {
	variable := ui.Variable{
		Handler: h,
	}

	r := s.router.PathPrefix("/variables").Subrouter()
	r.HandleFunc("", variable.Index).Methods("GET")
	r.HandleFunc("/create", variable.Create).Methods("GET")
	r.HandleFunc("", variable.Store).Methods("POST")
	r.HandleFunc("/{variable:[0-9]+}", variable.Destroy).Methods("DELETE")
	r.Use(mw.AuthVariable)
}

func (s *uiServer) initKey(h web.Handler, mw web.Middleware) {
	key := ui.Key{
		Handler: h,
	}

	r := s.router.PathPrefix("/keys").Subrouter()
	r.HandleFunc("", key.Index).Methods("GET")
	r.HandleFunc("/create", key.Create).Methods("GET")
	r.HandleFunc("", key.Store).Methods("POST")
	r.HandleFunc("/{key:[0-9]+}/edit", key.Edit).Methods("GET")
	r.HandleFunc("/{key:[0-9]+}", key.Update).Methods("PATCH")
	r.HandleFunc("/{key:[0-9]+}", key.Destroy).Methods("DELETE")
	r.Use(mw.AuthKey)
}

func (s *uiServer) init() {
	gob.Register(form.NewErrors())
	gob.Register(template.Alert{})
	gob.Register(make(map[string]string))

	s.router = mux.NewRouter()

	builds := model.BuildStore{
		DB: s.db,
	}

	objects := model.ObjectStore{
		DB: s.db,
	}

	users := model.UserStore{
		DB: s.db,
	}

	vars := model.VariableStore{
		DB: s.db,
	}

	keys := model.KeyStore{
		DB: s.db,
	}

	wh := web.New(securecookie.New(s.hash, s.key), session.New(s.client, s.key), users)
	mw := web.Middleware{
		Handler:   wh,
		Builds:    builds,
		Objects:   objects,
		Users:     users,
		Variables: vars,
		Keys:      keys,
	}

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
	s.initTag(wh, mw)
	s.initVariable(wh, mw)
	s.initKey(wh, mw)

	s.Server.Init(web.NewLog(web.NewSpoof(s.router)))
}
