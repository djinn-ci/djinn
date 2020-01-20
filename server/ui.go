package server

import (
	"encoding/gob"
	"net/http"

	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/web/ui"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type UI struct {
	Server

	csrf func(http.Handler) http.Handler

	artifact     ui.Artifact
	auth         ui.Auth
	build        ui.Build
	collaborator ui.Collaborator
	image        ui.Image
	invite       ui.Invite
	job          ui.Job
	key          ui.Key
	namespace    ui.Namespace
	oauth        ui.Oauth
	object       ui.Object
	repo         ui.Repo
	tag          ui.Tag
	user         ui.User
	variable     ui.Variable
	gate         gate
}

func (s *UI) Init() {
	s.Server.Init()

	gob.Register(form.NewErrors())
	gob.Register(template.Alert{})
	gob.Register(make(map[string]string))

	s.csrf = csrf.Protect(
		s.CSRFToken,
		csrf.RequestHeader("X-CSRF-Token"),
		csrf.FieldName("csrf_token"),
	)

	s.router = mux.NewRouter()

	s.router.NotFoundHandler = http.HandlerFunc(notFound)
	s.router.MethodNotAllowedHandler = http.HandlerFunc(methodNotAllowed)

//	assets := http.StripPrefix("/assets/", http.FileServer(http.Dir(s.Assets)))
//
//	s.router.PathPrefix("/assets/").Handler(assets)

	store := model.Store{
		DB: s.DB,
	}

	s.gate = gate{
		users:      model.UserStore{
			Store: store,
		},
		namespaces: model.NamespaceStore{
			Store: store,
		},
		images:     model.ImageStore{
			Store: store,
		},
		objects:    model.ObjectStore{
			Store: store,
		},
		variables:  model.VariableStore{
			Store: store,
		},
		keys:       model.KeyStore{
			Store: store,
		},
	}

	s.artifact = ui.Artifact{
		Handler:   s.Handler,
		FileStore: s.Artifacts,
	}

	s.auth = ui.Auth{
		Handler:   s.Handler,
		Providers: s.Providers,
	}

	s.build = ui.Build{
		Core: s.Server.build,
	}

	s.collaborator = ui.Collaborator{
		Core: s.Server.collaborator,
	}

	s.image = ui.Image{
		Core: s.Server.image,
	}

	s.invite = ui.Invite{
		Core: s.Server.invite,
	}

	s.job = ui.Job{
		Core: s.Server.job,
	}

	s.key = ui.Key{
		Core: s.Server.key,
	}

	s.oauth = ui.Oauth{
		Handler:   s.Handler,
		Providers: s.Providers,
	}

	s.object = ui.Object{
		Core: s.Server.object,
	}

	s.repo = ui.Repo{
		Handler:   s.Handler,
		Redis:     s.Redis,
		Providers: s.Providers,
	}

	s.tag = ui.Tag{
		Core: s.Server.tag,
	}

	s.user = ui.User{
		Handler:   s.Server.Handler,
		Invites:   model.InviteStore{
			Store: store,
		},
		Providers: s.Providers,
	}

	s.variable = ui.Variable{
		Core: s.Server.variable,
	}

	s.namespace = ui.Namespace{
		Core:     s.Server.namespace,
		Build:    s.build,
		Object:   s.object,
		Variable: s.variable,
		Key:      s.key,
	}
}

func (s *UI) Serve() error {
	s.Server.Server.Handler = web.NewSpoof(s.router)

	return s.Server.Serve()
}

func (s *UI) Hook() {
	r := s.router.PathPrefix("/hook").Subrouter()

	r.HandleFunc("/github", s.hook.Github)
	r.HandleFunc("/gitlab", s.hook.Gitlab)
}

func (s *UI) Auth() {
	r := s.router.PathPrefix("/").Subrouter()

	r.HandleFunc("/builds/create", s.build.Create).Methods("GET")
	r.HandleFunc("/builds", s.build.Store).Methods("POST")

	r.HandleFunc("/namespaces", s.namespace.Index).Methods("GET")
	r.HandleFunc("/namespaces/create", s.namespace.Create).Methods("GET")
	r.HandleFunc("/namespaces", s.namespace.Store).Methods("POST")

	r.HandleFunc("/settings", s.user.Settings).Methods("GET")
	r.HandleFunc("/settings/email", s.user.Email).Methods("PATCH")
	r.HandleFunc("/settings/password", s.user.Email).Methods("PATCH")
	r.HandleFunc("/settings/invites", s.user.Settings).Methods("GET")
	r.HandleFunc("/logout", s.auth.Logout).Methods("POST")

	r.HandleFunc("/invites/{invite:[0-9]+}", s.collaborator.Store).Methods("PATCH")
	r.HandleFunc("/invites/{invite:[0-9]+}", s.collaborator.Destroy).Methods("DELETE")

	r.HandleFunc("/repos", s.repo.Index).Methods("GET")
	r.HandleFunc("/repos/reload", s.repo.Reload).Methods("PATCH")
	r.HandleFunc("/repos/enable", s.repo.Store).Methods("POST")
	r.HandleFunc("/repos/disable/{repo:[0-9]+}", s.repo.Destroy).Methods("DELETE")

	r.HandleFunc("/", s.build.Index).Methods("GET")

	r.Use(s.Middleware.Auth, s.csrf)
}

func (s *UI) Oauth() {
	r := s.router.PathPrefix("/oauth").Subrouter()

	r.HandleFunc("/{provider}", s.oauth.Auth).Methods("GET")
	r.HandleFunc("/{provider}", s.oauth.Revoke).Methods("DELETE")

	r.Use(s.csrf)
}

func (s *UI) Guest() {
	r := s.router.PathPrefix("/").Subrouter()

	r.HandleFunc("/register", s.auth.Register).Methods("GET", "POST")
	r.HandleFunc("/login", s.auth.Login).Methods("GET", "POST")

	r.Use(s.Middleware.Guest, s.csrf)
}

func (s *UI) Namespace() {
	r := s.router.PathPrefix("/n/{username}/{namespace:[a-zA-Z0-9\\/?]+}").Subrouter()

	r.HandleFunc("", s.namespace.Show).Methods("GET")
	r.HandleFunc("/-/edit", s.namespace.Edit).Methods("GET")
	r.HandleFunc("/-/namespaces", s.namespace.Show).Methods("GET")
	r.HandleFunc("/-/objects", s.namespace.Show).Methods("GET")
	r.HandleFunc("/-/variables", s.namespace.Show).Methods("GET")
	r.HandleFunc("/-/keys", s.namespace.Show).Methods("GET")
	r.HandleFunc("/-/collaborators", s.namespace.Show).Methods("GET")
	r.HandleFunc("/-/collaborators", s.invite.Store).Methods("POST")
	r.HandleFunc("/-/collaborators/{collaborator}", s.collaborator.Destroy).Methods("DELETE")
	r.HandleFunc("", s.namespace.Update).Methods("PATCH")
	r.HandleFunc("", s.namespace.Destroy).Methods("DELETE")

	r.Use(s.Middleware.Gate(s.gate.namespace), s.csrf)
}

func (s *UI) Build() {
	r := s.router.PathPrefix("/b/{username}/{build:[0-9]+}").Subrouter()

	r.HandleFunc("", s.build.Show).Methods("GET")
	r.HandleFunc("", s.build.Kill).Methods("DELETE")
	r.HandleFunc("/manifest", s.build.Show).Methods("GET")
	r.HandleFunc("/manifest/raw", s.build.Show).Methods("GET")
	r.HandleFunc("/output/raw", s.build.Show).Methods("GET")
	r.HandleFunc("/objects", s.build.Show).Methods("GET")
	r.HandleFunc("/variables", s.build.Show).Methods("GET")
	r.HandleFunc("/keys", s.build.Show).Methods("GET")
	r.HandleFunc("/jobs/{job:[0-9]+}", s.job.Show).Methods("GET")
	r.HandleFunc("/jobs/{job:[0-9]+}/output/raw", s.job.Show).Methods("GET")
	r.HandleFunc("/artifacts", s.build.Show).Methods("GET")
	r.HandleFunc("/artifacts/{artifact:[0-9]+}/download/{name}", s.artifact.Show).Methods("GET")
	r.HandleFunc("/tags", s.build.Show).Methods("GET")
	r.HandleFunc("/tags", s.tag.Store).Methods("POST")
	r.HandleFunc("/tags/{tag:[0-9]+}", s.tag.Destroy).Methods("DELETE")

	r.Use(s.Middleware.Gate(s.gate.build), s.csrf)
}

func (s *UI) Image() {
	r := s.router.PathPrefix("/images").Subrouter()

	r.HandleFunc("/create", s.image.Create).Methods("GET")
	r.HandleFunc("/{image:[0-9]+}/download/{name}", s.image.Download).Methods("GET")
	r.HandleFunc("/{image:[0-9]+}", s.image.Destroy).Methods("DELETE")
	r.HandleFunc("", s.image.Store).Methods("POST")
	r.HandleFunc("", s.image.Index).Methods("GET")

	r.Use(s.Middleware.Auth, s.Middleware.Gate(s.gate.image), s.csrf)
}

func (s *UI) Object() {
	r := s.router.PathPrefix("/objects").Subrouter()

	r.HandleFunc("/create", s.object.Create).Methods("GET")
	r.HandleFunc("", s.object.Index).Methods("GET")
	r.HandleFunc("/{object:[0-9]+}", s.object.Show).Methods("GET")
	r.HandleFunc("/{object:[0-9]+}/download/{name}", s.object.Download).Methods("GET")
	r.HandleFunc("/{object:[0-9]+}", s.object.Destroy).Methods("DELETE")
	r.HandleFunc("", s.object.Store).Methods("POST")
	r.HandleFunc("", s.object.Index).Methods("GET")

	r.Use(s.Middleware.Gate(s.gate.object), s.csrf)
}

func (s *UI) Variable() {
	r := s.router.PathPrefix("/variables").Subrouter()

	r.HandleFunc("/create", s.variable.Create).Methods("GET")
	r.HandleFunc("/{variable:[0-9]+}", s.variable.Destroy).Methods("DELETE")
	r.HandleFunc("", s.variable.Store).Methods("POST")
	r.HandleFunc("", s.variable.Index).Methods("GET")

	r.Use(s.Middleware.Gate(s.gate.variable), s.csrf)
}

func (s *UI) Key() {
	r := s.router.PathPrefix("/keys").Subrouter()

	r.HandleFunc("/create", s.key.Create).Methods("GET")
	r.HandleFunc("/{key:[0-9]+}/edit", s.key.Edit).Methods("GET")
	r.HandleFunc("/{key:[0-9]+}", s.key.Update).Methods("PATCH")
	r.HandleFunc("/{key:[0-9]+}", s.key.Destroy).Methods("DELETE")
	r.HandleFunc("", s.key.Store).Methods("POST")
	r.HandleFunc("", s.key.Index).Methods("GET")

	r.Use(s.Middleware.Gate(s.gate.key), s.csrf)
}
