package main

import (
	"encoding/gob"
	"net/http"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/model/query"
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

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	web.HTMLError(w, "Not found", http.StatusNotFound)
}

func methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	web.HTMLError(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	resources := model.ResourceStore{
		Store: store,
	}

	m := model.Model{
		DB: s.db,
	}

	resources.Register("build", model.Type{
		Table:    "builds",
		Resource: &model.Build{
			Model: m,
		},
		HandleFind: func(name string, vars map[string]string) []query.Option {
			username := vars["username"]
			id, _ := strconv.ParseInt(vars[name], 10, 64)

			return []query.Option{
				query.Columns("*"),
				query.Table("builds"),
				query.WhereEqQuery("user_id",
					query.Select(
						query.Columns("id"),
						query.Table("users"),
						query.WhereEq("username", username),
					),
				),
				query.WhereEq("id", id),
			}
		},
	})

	resources.Register("object", model.Type{
		Table:    "objects",
		Resource: &model.Object{
			Model: m,
		},
	})

	resources.Register("variable", model.Type{
		Table:    "variables",
		Resource: &model.Variable{
			Model: m,
		},
	})

	resources.Register("key", model.Type{
		Table:    "keys",
		Resource: &model.Key{
			Model: m,
		},
	})

	resources.Register("namespace", model.Type{
		Table:    "namespaces",
		Resource: &model.Namespace{
			Model: m,
		},
		HandleFind: func(name string, vars map[string]string) []query.Option {
			username := vars["username"]
			path := strings.TrimSuffix(vars[name], "/")

			return []query.Option{
				query.Columns("*"),
				query.Table("namespaces"),
				query.WhereEqQuery("user_id",
					query.Select(
						query.Columns("id"),
						query.Table("users"),
						query.WhereEq("username", username),
					),
				),
				query.WhereEq("path", path),
			}
		},
	})

	wh := web.Handler{
		Store:        session.New(s.client, s.key),
		SecureCookie: securecookie.New(s.hash, s.key),
		Users:        users,
	}

	mw := web.Middleware{
		Handler:   wh,
		Resources: resources,
	}

	s.router.NotFoundHandler = http.HandlerFunc(notFoundHandler)
	s.router.MethodNotAllowedHandler = http.HandlerFunc(methodNotAllowedHandler)

	assets := http.StripPrefix("/assets/", http.FileServer(http.Dir(s.assets)))

	s.router.PathPrefix("/assets/").Handler(assets)

	auth := ui.Auth{
		Handler: wh,
	}

	build := ui.Build{
		Handler: wh,
		Queues:  s.Queues,
	}

	namespace := ui.Namespace{
		Handler: wh,
	}

	object := ui.Object{
		Handler:   wh,
		FileStore: s.objects,
		Limit:     s.limit,
	}

	variable := ui.Variable{
		Handler: wh,
	}

	key := ui.Key{
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

	authRouter.HandleFunc("/logout", auth.Logout).Methods("POST")

	authRouter.Use(mw.Auth)

	namespaceRouter := s.router.PathPrefix("/n/{username}/{namespace:[a-zA-Z0-9\\/?]+}").Subrouter()
	namespaceRouter.HandleFunc("", namespace.Show).Methods("GET")
	namespaceRouter.HandleFunc("/-/edit", namespace.Edit).Methods("GET")
	namespaceRouter.HandleFunc("/-/namespaces", namespace.Show).Methods("GET")
	namespaceRouter.HandleFunc("/-/objects", namespace.Show).Methods("GET")
	namespaceRouter.HandleFunc("/-/variables", namespace.Show).Methods("GET")
	namespaceRouter.HandleFunc("/-/keys", namespace.Show).Methods("GET")
	namespaceRouter.HandleFunc("", namespace.Update).Methods("PATCH")
	namespaceRouter.HandleFunc("", namespace.Destroy).Methods("DELETE")
	namespaceRouter.Use(mw.GateResource("namespace"))

	buildRouter := s.router.PathPrefix("/b/{username}/{build:[0-9]+}").Subrouter()
	buildRouter.HandleFunc("", build.Show).Methods("GET")
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
	buildRouter.Use(mw.GateResource("build"))

	objectRouter := s.router.PathPrefix("/objects").Subrouter()
	objectRouter.HandleFunc("", object.Index).Methods("GET")
	objectRouter.HandleFunc("/{object:[0-9]+}", object.Show).Methods("GET")
	objectRouter.HandleFunc("/{object:[0-9]+}/download/{name}", object.Download).Methods("GET")
	objectRouter.HandleFunc("/{object:[0-9]+}", object.Destroy).Methods("DELETE")
	objectRouter.Use(mw.GateResource("object"))

	variableRouter := s.router.PathPrefix("/variables").Subrouter()
	variableRouter.HandleFunc("/{variable:[0-9]+}", variable.Destroy).Methods("DELETE")
	variableRouter.Use(mw.GateResource("variable"))

	keyRouter := s.router.PathPrefix("/keys").Subrouter()
	keyRouter.HandleFunc("/{key:[0-9]+}/edit", key.Edit).Methods("GET")
	keyRouter.HandleFunc("/{key:[0-9]+}", key.Update).Methods("PATCH")
	keyRouter.HandleFunc("/{key:[0-9]+}", key.Destroy).Methods("DELETE")
	keyRouter.Use(mw.GateResource("key"))

	s.Server.Init(web.NewSpoof(s.router))
}
