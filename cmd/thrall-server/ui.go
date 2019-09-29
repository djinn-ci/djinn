package main

import (
	"encoding/gob"
	nethttp "net/http"

	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/web/ui"
	"github.com/andrewpillar/thrall/http"
	"github.com/andrewpillar/thrall/session"
	"github.com/andrewpillar/thrall/template"

	"github.com/jmoiron/sqlx"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/mux"

	"github.com/go-redis/redis"

	"github.com/RichardKnop/machinery/v1"
)

type server struct {
	*http.Server

	db     *sqlx.DB
	client *redis.Client

	objects   filestore.FileStore
	artifacts filestore.FileStore

	drivers map[string]struct{}

	hash   []byte
	key    []byte
	limit  int64

	queue  *machinery.Server
	router *mux.Router
}

type uiServer struct {
	server

	assets string
}

func notFoundHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	web.HTMLError(w, "Not found", nethttp.StatusNotFound)
}

func methodNotAllowedHandler(w nethttp.ResponseWriter, r *nethttp.Request) {
	web.HTMLError(w, "Method not allowed", nethttp.StatusMethodNotAllowed)
}

func (s *uiServer) initAuth(h web.Handler, mw web.Middleware) {
	auth := ui.Auth{
		Handler: h,
	}

	s.router.HandleFunc("/register", auth.Register).Methods("GET", "POST")
	s.router.HandleFunc("/login", auth.Login).Methods("GET", "POST")
	s.router.HandleFunc("/logout", auth.Logout).Methods("POST")
}

func (s *uiServer) init() {
	gob.Register(form.NewErrors())
	gob.Register(template.Alert{})
	gob.Register(make(map[string]string))

	s.router = mux.NewRouter()

	store := model.Store{
		DB: s.db,
	}

	users := model.UserStore{
		Store: store,
	}

	namespaces := model.NamespaceStore{
		Store: store,
	}

	builds := model.BuildStore{
		Store: store,
	}

	wh := web.Handler{
		Store:        session.New(s.client, s.key),
		SecureCookie: securecookie.New(s.hash, s.key),
		Users:        users,
	}

	mw := web.Middleware{
		Handler:   wh,
	}

	s.router.NotFoundHandler = nethttp.HandlerFunc(notFoundHandler)
	s.router.MethodNotAllowedHandler = nethttp.HandlerFunc(methodNotAllowedHandler)

	assets := nethttp.StripPrefix("/assets/", nethttp.FileServer(nethttp.Dir(s.assets)))

	s.router.PathPrefix("/assets/").Handler(assets)

	auth := ui.Auth{
		Handler: wh,
	}

	build := ui.Build{
		Handler: wh,
		Client:  s.client,
		Builds:  builds,
		Drivers: s.drivers,
		Queue:   s.queue,
	}

	object := ui.Object{
		Handler:   wh,
		Limit:     s.limit,
		Objects:   model.ObjectStore{
			Store: store,
		},
		FileStore: s.objects,
	}

	variable := ui.Variable{
		Handler:   wh,
		Variables: model.VariableStore{
			Store: store,
		},
	}

	key := ui.Key{
		Handler: wh,
		Keys:    model.KeyStore{
			Store: store,
		},
	}

	namespace := ui.Namespace{
		Handler:    wh,
		Namespaces: namespaces,
		Build:      build,
		Object:     object,
		Variable:   variable,
		Key:        key,
	}

	collaborator := ui.Collaborator{
		Handler: wh,
		Invites: model.InviteStore{
			Store: store,
		},
	}

	invite := ui.Invite{
		Handler: wh,
	}

	job := ui.Job{
		Handler: wh,
	}

	artifact := ui.Artifact{
		Handler: wh,
	}

	tag := ui.Tag{
		Handler: wh,
	}

	user := ui.User{
		Handler: wh,
		Invites: model.InviteStore{
			Store: store,
		},
	}

	guestRouter := s.router.PathPrefix("/").Subrouter()
	guestRouter.HandleFunc("/register", auth.Register).Methods("GET", "POST")
	guestRouter.HandleFunc("/login", auth.Login).Methods("GET", "POST")
	guestRouter.Use(mw.Guest)

	authRouter := s.router.PathPrefix("/").Subrouter()

	authRouter.HandleFunc("/", build.Index).Methods("GET", "POST")
	authRouter.HandleFunc("/builds/create", build.Create).Methods("GET")
	authRouter.HandleFunc("/builds", build.Store).Methods("POST")

	authRouter.HandleFunc("/namespaces", namespace.Index).Methods("GET")
	authRouter.HandleFunc("/namespaces/create", namespace.Create).Methods("GET")
	authRouter.HandleFunc("/namespaces", namespace.Store).Methods("POST")

	authRouter.HandleFunc("/objects", object.Index).Methods("GET")
	authRouter.HandleFunc("/objects/create", object.Create).Methods("GET")
	authRouter.HandleFunc("/objects", object.Store).Methods("POST")

	authRouter.HandleFunc("/variables", variable.Index).Methods("GET")
	authRouter.HandleFunc("/variables/create", variable.Create).Methods("GET")
	authRouter.HandleFunc("/variables", variable.Store).Methods("POST")

	authRouter.HandleFunc("/keys", key.Index).Methods("GET")
	authRouter.HandleFunc("/keys/create", key.Create).Methods("GET")
	authRouter.HandleFunc("/keys", key.Store).Methods("POST")

	authRouter.HandleFunc("/settings", user.Settings).Methods("GET")
	authRouter.HandleFunc("/settings/invites", user.Settings).Methods("GET")
	authRouter.HandleFunc("/logout", auth.Logout).Methods("POST")

	authRouter.HandleFunc("/invites/{invite:[0-9]+}", collaborator.Store).Methods("PATCH")
	authRouter.HandleFunc("/invites/{invite:[0-9]+}", invite.Destroy).Methods("DELETE")

	authRouter.Use(mw.Auth)

	gate := gate{
		users:      users,
		namespaces: namespaces,
		objects:    model.ObjectStore{
			Store: store,
		},
		variables:  model.VariableStore{
			Store: store,
		},
		keys:      model.KeyStore{
			Store: store,
		},
	}

	namespaceRouter := s.router.PathPrefix("/n/{username}/{namespace:[a-zA-Z0-9\\/?]+}").Subrouter()
	namespaceRouter.HandleFunc("", namespace.Show).Methods("GET")
	namespaceRouter.HandleFunc("/-/edit", namespace.Edit).Methods("GET")
	namespaceRouter.HandleFunc("/-/namespaces", namespace.Show).Methods("GET")
	namespaceRouter.HandleFunc("/-/objects", namespace.Show).Methods("GET")
	namespaceRouter.HandleFunc("/-/variables", namespace.Show).Methods("GET")
	namespaceRouter.HandleFunc("/-/keys", namespace.Show).Methods("GET")
	namespaceRouter.HandleFunc("/-/collaborators", namespace.Show).Methods("GET")
	namespaceRouter.HandleFunc("/-/collaborators", invite.Store).Methods("POST")
	namespaceRouter.HandleFunc("/-/collaborators/{collaborator}", collaborator.Destroy).Methods("DELETE")
	namespaceRouter.HandleFunc("", namespace.Update).Methods("PATCH")
	namespaceRouter.HandleFunc("", namespace.Destroy).Methods("DELETE")
	namespaceRouter.Use(mw.Gate(gate.namespace))

	buildRouter := s.router.PathPrefix("/b/{username}/{build:[0-9]+}").Subrouter()
	buildRouter.HandleFunc("", build.Show).Methods("GET")
	buildRouter.HandleFunc("", build.Kill).Methods("DELETE")
	buildRouter.HandleFunc("/manifest", build.Show).Methods("GET")
	buildRouter.HandleFunc("/manifest/raw", build.Show).Methods("GET")
	buildRouter.HandleFunc("/output", build.Show).Methods("GET")
	buildRouter.HandleFunc("/output/raw", build.Show).Methods("GET")
	buildRouter.HandleFunc("/objects", build.Show).Methods("GET")
	buildRouter.HandleFunc("/variables", build.Show).Methods("GET")
	buildRouter.HandleFunc("/jobs/{job:[0-9]+}", job.Show).Methods("GET")
	buildRouter.HandleFunc("/jobs/{job:[0-9]+}/output/raw", job.Show).Methods("GET")
	buildRouter.HandleFunc("/artifacts", build.Show).Methods("GET")
	buildRouter.HandleFunc("/artifacts/{artifact:[0-9]+}/download/{name}", artifact.Show).Methods("GET")
	buildRouter.HandleFunc("/tags", build.Show).Methods("GET")
	buildRouter.HandleFunc("/tags", tag.Store).Methods("POST")
	buildRouter.HandleFunc("/tags/{tag:[0-9]+}", tag.Destroy).Methods("DELETE")
	buildRouter.Use(mw.Gate(gate.build))

	objectRouter := s.router.PathPrefix("/objects").Subrouter()
	objectRouter.HandleFunc("", object.Index).Methods("GET")
	objectRouter.HandleFunc("/{object:[0-9]+}", object.Show).Methods("GET")
	objectRouter.HandleFunc("/{object:[0-9]+}/download/{name}", object.Download).Methods("GET")
	objectRouter.HandleFunc("/{object:[0-9]+}", object.Destroy).Methods("DELETE")
	objectRouter.Use(mw.Gate(gate.object))

	variableRouter := s.router.PathPrefix("/variables").Subrouter()
	variableRouter.HandleFunc("/{variable:[0-9]+}", variable.Destroy).Methods("DELETE")
	variableRouter.Use(mw.Gate(gate.variable))

	keyRouter := s.router.PathPrefix("/keys").Subrouter()
	keyRouter.HandleFunc("/{key:[0-9]+}/edit", key.Edit).Methods("GET")
	keyRouter.HandleFunc("/{key:[0-9]+}", key.Update).Methods("PATCH")
	keyRouter.HandleFunc("/{key:[0-9]+}", key.Destroy).Methods("DELETE")
	keyRouter.Use(mw.Gate(gate.key))

	s.Server.Init(web.NewSpoof(s.router))
}
