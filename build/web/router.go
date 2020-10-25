package web

import (
	"context"
	"net/http"
	"strconv"

	"github.com/andrewpillar/djinn/block"
	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/build/handler"
	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/object"
	"github.com/andrewpillar/djinn/provider"
	"github.com/andrewpillar/djinn/server"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/variable"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"

	"github.com/RichardKnop/machinery/v1"
)

// Router is what registers the UI and API routes for managing builds. It
// implements the server.Router interface.
type Router struct {
	build    handler.Build
	job      handler.Job
	tag      handler.Tag
	hook     handler.Hook
	artifact handler.ArtifactAPI

	// Redis is the client connection to redis, used for handling build
	// submission and killing.
	Redis *redis.Client

	// Block is the cipher block used for decrypting API access tokens needed
	// when consuming a webhook from a provider.
	Block *crypto.Block

	// Hasher is the hashing mechanism to use when generating hashes for build
	// artifacts.
	Hasher *crypto.Hasher

	// Registry holds the registered provider.Client implementations. This is
	// used during the consumption of webhooks to interface with their APIs.
	Registry *provider.Registry

	// Middleware is the middleware that is applied to any routes registered
	// from this router.
	Middleware web.Middleware

	// Artifacts is the storage mechanism used for storing artifacts. This used
	// for downloading artifacts from the server.
	Artifacts block.Store

	// Queues are the different queues builds can be submitted onto based on the
	// driver being used for the build.
	Queues map[string]*machinery.Server
}

var _ server.Router = (*Router)(nil)

// Gate returns a web.Gate that checks if the current authenticated User has
// the access permissions to the current Build. If the current user can access
// the current build then it is set in the request's context.
func Gate(db *sqlx.DB) web.Gate {
	users := user.NewStore(db)
	namespaces := namespace.NewStore(db)

	return func(u *user.User, r *http.Request) (*http.Request, bool, error) {
		var ok bool

		switch r.Method {
		case "GET":
			_, ok = u.Permissions["build:read"]
		case "POST", "PATCH":
			_, ok = u.Permissions["build:write"]
		case "DELETE":
			_, ok = u.Permissions["build:delete"]
		}

		base := web.BasePath(r.URL.Path)

		// Are we creating a build or viewing a list of builds.
		if base == "/" || base == "create" || base == "builds" {
			return r, ok, nil
		}

		if base == "tags" && r.Method == "DELETE" {
			return r, ok, nil
		}

		vars := mux.Vars(r)

		owner, err := users.Get(query.Where("username", "=", vars["username"]))

		if err != nil {
			return r, false, errors.Err(err)
		}

		id, _ := strconv.ParseInt(vars["build"], 10, 64)

		b, err := build.NewStore(db, owner).Get(query.Where("id", "=", id))

		if err != nil {
			return r, false, errors.Err(err)
		}

		if b.IsZero() {
			return r, false, nil
		}

		r = r.WithContext(context.WithValue(r.Context(), "build", b))

		if !b.NamespaceID.Valid {
			return r, ok && u.ID == b.UserID, nil
		}

		root, err := namespaces.Get(
			query.WhereQuery("root_id", "=", namespace.SelectRootID(b.NamespaceID.Int64)),
			query.WhereQuery("id", "=", namespace.SelectRootID(b.NamespaceID.Int64)),
		)

		if err != nil {
			return r, false, errors.Err(err)
		}

		cc, err := namespace.NewCollaboratorStore(db, root).All()

		if err != nil {
			return r, false, errors.Err(err)
		}

		root.LoadCollaborators(cc)
		return r, root.AccessibleBy(u), nil
	}
}

// Init intialises the primary handler.Build for handling the primary logic
// of Build submission and management. This will setup the database.Loader for
// relationship loading, and the related database stores. The exported
// properties on the Router itself are passed through to the underlying
// handler.Build.
func (r *Router) Init(h web.Handler) {
	namespaces := namespace.NewStore(h.DB)
	tags := build.NewTagStore(h.DB)
	triggers := build.NewTriggerStore(h.DB)
	stages := build.NewStageStore(h.DB)
	artifacts := build.NewArtifactStore(h.DB)

	loaders := database.NewLoaders()
	loaders.Put("user", h.Users)
	loaders.Put("namespace", namespaces)
	loaders.Put("build_tag", tags)
	loaders.Put("build_trigger", triggers)
	loaders.Put("build_stage", stages)
	loaders.Put("build_artifact", artifacts)

	r.build = handler.Build{
		Handler:   h,
		Loaders:   loaders,
		Objects:   object.NewStore(h.DB),
		Variables: variable.NewStore(h.DB),
		Client:    r.Redis,
		Hasher:    r.Hasher,
		Queues:    r.Queues,
	}
	r.job = handler.Job{
		Handler: h,
		Loaders: loaders,
	}
	r.tag = handler.Tag{
		Handler: h,
	}
	r.hook = handler.Hook{
		Build:     r.build,
		Block:     r.Block,
		Repos:     provider.NewRepoStore(h.DB),
		Providers: provider.NewStore(h.DB),
		Registry:  r.Registry,
	}
	r.artifact = handler.ArtifactAPI{
		Handler: h,
		Loaders: loaders,
	}
}

// RegisterUI registers the UI routes for working with builds. There are three
// types of route groups, webhooks, simple auth routes, and individual build
// routes. These routes, aside for webhook routes, respond with a "text/html"
// Content-Type.
//
// webhooks - The webhook routes are registered under the "/hook" prefix of the
// given router. No CSRF protection is applied to these routes, verficiation of
// the requests are done within the handlers themselves.
//
// simple auth routes - These routes are registered under the "/" prefix of the
// given router. The Auth middleware is applied to all registered routes. CSRF
// protection is applied to all the registered routes.
//
// individual build routes - These routes are registered under the
// "/b/{username}/{build:[0-9]}" prefix of the given router. Each given gate
// is applied to the registered routes, along with the given CSRF protection.
func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	build := handler.UI{
		Build:     r.build,
		Artifacts: r.Artifacts,
	}

	tag := handler.TagUI{Tag: r.tag}
	job := handler.JobUI{Job: r.job}

	hookRouter := mux.PathPrefix("/hook").Subrouter()
	hookRouter.HandleFunc("/github", r.hook.GitHub).Methods("POST")
	hookRouter.HandleFunc("/gitlab", r.hook.GitLab).Methods("POST")

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/", build.Index).Methods("GET")
	auth.HandleFunc("/builds/create", build.Create).Methods("GET")
	auth.HandleFunc("/builds", build.Store).Methods("POST")
	auth.Use(r.Middleware.Auth, r.Middleware.Gate(gates...), csrf)

	sr := mux.PathPrefix("/b/{username}/{build:[0-9]+}").Subrouter()
	sr.HandleFunc("", build.Show).Methods("GET")
	sr.HandleFunc("", build.Destroy).Methods("DELETE")
	sr.HandleFunc("/manifest", build.Show).Methods("GET")
	sr.HandleFunc("/manifest/raw", build.Show).Methods("GET")
	sr.HandleFunc("/output/raw", build.Show).Methods("GET")
	sr.HandleFunc("/objects", build.Show).Methods("GET")
	sr.HandleFunc("/variables", build.Show).Methods("GET")
	sr.HandleFunc("/keys", build.Show).Methods("GET")
	sr.HandleFunc("/jobs/{job:[0-9]+}", job.Show).Methods("GET")
	sr.HandleFunc("/jobs/{job:[0-9]+}/output/raw", job.Show).Methods("GET")
	sr.HandleFunc("/artifacts", build.Show).Methods("GET")
	sr.HandleFunc("/artifacts/{artifact:[0-9]+}/download/{name}", build.Download).Methods("GET")
	sr.HandleFunc("/tags", build.Show).Methods("GET")
	sr.HandleFunc("/tags", tag.Store).Methods("POST")
	sr.HandleFunc("/tags/{tag:[0-9]+}", tag.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...), csrf)
}

// RegisterAPI registers the API routes for working with builds. The given
// prefix string is used to specify where the API is being served under. This
// applies all of the given gates to all routes registered. These routes
// response with a "application/json" Content-Type.
func (r *Router) RegisterAPI(prefix string, mux *mux.Router, gates ...web.Gate) {
	build := handler.API{
		Prefix: prefix,
		Build:  r.build,
	}

	job := handler.JobAPI{
		Prefix: prefix,
		Job:    r.job,
	}

	tag := handler.TagAPI{
		Prefix: prefix,
		Tag:    r.tag,
	}

	r.artifact.Prefix = prefix

	auth := mux.PathPrefix("/builds").Subrouter()
	auth.HandleFunc("", build.Index).Methods("GET", "HEAD")
	auth.HandleFunc("", build.Store).Methods("POST")
	auth.Use(r.Middleware.Gate(gates...))

	sr := mux.PathPrefix("/b/{username}/{build:[0-9]+}").Subrouter()
	sr.HandleFunc("", build.Show).Methods("GET")
	sr.HandleFunc("", build.Destroy).Methods("DELETE")
	sr.HandleFunc("/objects", build.Show).Methods("GET")
	sr.HandleFunc("/variables", build.Show).Methods("GET")
	sr.HandleFunc("/keys", build.Show).Methods("GET")
	sr.HandleFunc("/jobs", job.Index).Methods("GET")
	sr.HandleFunc("/jobs/{job:[0-9]+}", job.Show).Methods("GET")
	sr.HandleFunc("/artifacts", r.artifact.Index).Methods("GET")
	sr.HandleFunc("/artifacts/{artifact:[0-9]+}", r.artifact.Show).Methods("GET")
	sr.HandleFunc("/tags", tag.Index).Methods("GET")
	sr.HandleFunc("/tags", tag.Store).Methods("POST")
	sr.HandleFunc("/tags/{tag:[0-9]+}", tag.Show).Methods("GET")
	sr.HandleFunc("/tags/{tag:[0-9]+}", tag.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...))
}
