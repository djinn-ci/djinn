package router

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"djinn-ci.com/build"
	"djinn-ci.com/build/handler"
	"djinn-ci.com/config"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	"djinn-ci.com/web"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

// Router is what registers the UI and API routes for managing builds. It
// implements the server.Router interface.
type Router struct {
	middleware web.Middleware

	build    handler.Build
	job      handler.Job
	tag      handler.Tag
	hook     handler.Hook
	artifact handler.Artifact
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

		isJson := strings.HasPrefix(r.Header.Get("Accept"), "application/json") ||
			strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")

		base := webutil.BasePath(r.URL.Path)

		// Are we creating a build or viewing a list of builds.
		if base == "/" || base == "create" || base == "builds" {
			return r, ok, nil
		}

		vars := mux.Vars(r)

		owner, err := users.Get(query.Where("username", "=", query.Arg(vars["username"])))

		if err != nil {
			return r, false, errors.Err(err)
		}

		if owner.IsZero() {
			return r, false, nil
		}

		id, _ := strconv.ParseInt(vars["build"], 10, 64)

		b, err := build.NewStore(db, owner).Get(query.Where("number", "=", query.Arg(id)))

		if err != nil {
			return r, false, errors.Err(err)
		}

		if b.IsZero() {
			return r, false, nil
		}

		r = r.WithContext(context.WithValue(r.Context(), "build", b))

		if !b.NamespaceID.Valid {
			return r, ok && owner.ID == u.ID, nil
		}

		root, err := namespaces.Get(
			query.Where("root_id", "=", namespace.SelectRootID(b.NamespaceID.Int64)),
			query.Where("id", "=", namespace.SelectRootID(b.NamespaceID.Int64)),
		)

		if err != nil {
			return r, false, errors.Err(err)
		}

		cc, err := namespace.NewCollaboratorStore(db, root).All()

		if err != nil {
			return r, false, errors.Err(err)
		}

		root.LoadCollaborators(cc)

		if isJson {
			return r, ok && root.AccessibleBy(u), nil
		}

		// Account for public namespaces.
		return r, root.AccessibleBy(u), nil
	}
}

func New(cfg *config.Server, h web.Handler, mw web.Middleware) *Router {
	build := handler.New(h, cfg.Artifacts().Store, cfg.Redis(), cfg.Hasher(), cfg.Producers())

	return &Router{
		middleware: mw,
		build:      build,
		job:        handler.NewJob(h),
		tag:        handler.NewTag(h),
		hook:       handler.NewHook(build, cfg.BlockCipher(), cfg.Providers()),
		artifact:   handler.NewArtifact(h),
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
	build := handler.UI{Build: r.build}
	tag := handler.TagUI{Tag: r.tag}
	job := handler.JobUI{Job: r.job}

	hookRouter := mux.PathPrefix("/hook").Subrouter()
	hookRouter.HandleFunc("/github", r.hook.GitHub).Methods("POST")
	hookRouter.HandleFunc("/gitlab", r.hook.GitLab).Methods("POST")

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/builds", build.Index).Methods("GET")
	auth.HandleFunc("/builds/create", build.Create).Methods("GET")
	auth.HandleFunc("/builds", build.Store).Methods("POST")
	auth.Use(r.middleware.Auth, r.middleware.Gate(gates...), csrf)

	sr := mux.PathPrefix("/b/{username}/{build:[0-9]+}").Subrouter()
	sr.HandleFunc("", build.Show).Methods("GET")
	sr.HandleFunc("", build.Destroy).Methods("DELETE")
	sr.HandleFunc("/manifest", build.Show).Methods("GET")
	sr.HandleFunc("/manifest/raw", build.Show).Methods("GET")
	sr.HandleFunc("/output/raw", build.Show).Methods("GET")
	sr.HandleFunc("/objects", build.Show).Methods("GET")
	sr.HandleFunc("/variables", build.Show).Methods("GET")
	sr.HandleFunc("/keys", build.Show).Methods("GET")
	sr.HandleFunc("/jobs/{name}", job.Show).Methods("GET")
	sr.HandleFunc("/jobs/{name}/output/raw", job.Show).Methods("GET")
	sr.HandleFunc("/artifacts", build.Show).Methods("GET")
	sr.HandleFunc("/artifacts/{name}", build.Download).Methods("GET")
	sr.HandleFunc("/tags", build.Show).Methods("GET")
	sr.HandleFunc("/tags", tag.Store).Methods("POST")
	sr.HandleFunc("/tags/{name}", tag.Destroy).Methods("DELETE")
	sr.Use(r.middleware.Gate(gates...), csrf)
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
	auth.Use(r.middleware.Gate(gates...))

	sr := mux.PathPrefix("/b/{username}/{build:[0-9]+}").Subrouter()
	sr.HandleFunc("", build.Show).Methods("GET")
	sr.HandleFunc("", build.Destroy).Methods("DELETE")
	sr.HandleFunc("/objects", build.Show).Methods("GET")
	sr.HandleFunc("/variables", build.Show).Methods("GET")
	sr.HandleFunc("/keys", build.Show).Methods("GET")
	sr.HandleFunc("/jobs", job.Index).Methods("GET")
	sr.HandleFunc("/jobs/{job:[0-9]+}", job.Show).Methods("GET")
	sr.HandleFunc("/artifacts", r.artifact.Index).Methods("GET")
	sr.HandleFunc("/artifacts/{name}", r.artifact.Show).Methods("GET")
	sr.HandleFunc("/tags", tag.Index).Methods("GET")
	sr.HandleFunc("/tags", tag.Store).Methods("POST")
	sr.HandleFunc("/tags/{name}", tag.Show).Methods("GET")
	sr.HandleFunc("/tags/{name}", tag.Destroy).Methods("DELETE")
	sr.Use(r.middleware.Gate(gates...))
}
