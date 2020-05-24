package web

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/namespace/handler"
	"github.com/andrewpillar/thrall/server"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

type Router struct {
	namespace    handler.Namespace
	invite       handler.Invite
	collaborator handler.Collaborator

	Middleware web.Middleware
}

var _ server.Router = (*Router)(nil)

// Gate returns a web.Gate that checks if the current authenticated User has
// the access permissions to the current Namespace, or if they are a
// Collaborator in that Namespace. If the current User can access the current
// Namespace, then it is set in the request's context.
func Gate(db *sqlx.DB) web.Gate {
	users := user.NewStore(db)
	namespaces := namespace.NewStore(db)

	ownerPaths := map[string]struct{}{
		"edit":          {},
		"collaborators": {},
	}

	ownerMethods := map[string]struct{}{
		"POST":   {},
		"PATCH":  {},
		"DELETE": {},
	}

	return func(u *user.User, r *http.Request) (*http.Request, bool, error) {
		var ok bool

		switch r.Method {
		case "GET":
			_, ok = u.Permissions["namespace:read"]
		case "POST", "PATCH":
			_, ok = u.Permissions["namespace:write"]
		case "DELETE":
			_, ok = u.Permissions["namespace:delete"]
		}

		if !ok {
			return r, false, nil
		}

		vars := mux.Vars(r)

		owner, err := users.Get(query.Where("username", "=", vars["username"]))

		if err != nil {
			return r, false, errors.Err(err)
		}

		path := strings.TrimSuffix(vars["namespace"], "/")

		n, err := namespace.NewStore(db, owner).Get(query.Where("path", "=", path))

		if err != nil {
			return r, false, errors.Err(err)
		}

		if n.IsZero() {
			return r, false, errors.Err(err)
		}

		// Can the current user modify/delete the current namespace.
		if _, ok := ownerMethods[r.Method]; ok {
			if owner.ID != u.ID {
				return r, false, nil
			}
		}

		if _, ok := ownerPaths[filepath.Base(r.URL.Path)]; ok {
			if owner.ID != u.ID {
				return r, false, nil
			}
		}

		r = r.WithContext(context.WithValue(r.Context(), "namespace", n))

		root, err := namespaces.Get(
			query.WhereQuery("root_id", "=", namespace.SelectRootID(n.ID)),
			query.WhereQuery("id", "=", namespace.SelectRootID(n.ID)),
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

// Init initialiases the primary handle.namespace for handling the primary
// logic of Namespace creation and management. This will setup the model.Loader
// for relationship loading, and the related model stores. The exported
// properties on the Router itself are pased through to the underlying
// handler.Namspace.
func (r *Router) Init(h web.Handler) {
	namespaces := namespace.NewStore(h.DB)

	loaders := model.NewLoaders()
	loaders.Put("user", h.Users)
	loaders.Put("namespace", namespaces)
	loaders.Put("build_tag", build.NewTagStore(h.DB))
	loaders.Put("build_trigger", build.NewTriggerStore(h.DB))

	r.namespace = handler.Namespace{
		Handler:    h,
		Loaders:    loaders,
		Builds:     build.NewStore(h.DB),
		Namespaces: namespaces,
	}

	r.invite = handler.Invite{
		Handler: h,
		Loaders: loaders,
	}

	r.collaborator = handler.Collaborator{
		Handler:    h,
		Invites:    namespace.NewInviteStore(h.DB),
		Namespaces: namespaces,
	}
}

// RegisterUI registers the UI routes for Namespace creation, and management.
// There are two types of routes, simple auth routes, and individual namespace
// routes. These routes respond with a text/html Content-Type.
//
// simple auth routes - These routes (/namespaces, /namespaces/create,
// /settings/invites, /invites/{invite:[0-9]+}) havbe the auth middleware
// applied to them to check if a user is authenticated to access the route. The
// given http.Handler is applied to these routes for CSRF protection.
//
// individual namespace routes - These routes (prefixed with
// /n/{username}/{namespace:[a-zA-Z0-9\\/?]+}), use the given http.Handler for
// CSRF protection, and the given gates for auth checks, and permission checks.
func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	namespace := handler.UI{
		Namespace: r.namespace,
		Invite:    handler.InviteUI{
			Invite: r.invite,
		},
		Collaborator: handler.CollaboratorUI{
			Collaborator: r.collaborator,
		},
	}

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/namespaces", namespace.Index).Methods("GET")
	auth.HandleFunc("/namespaces/create", namespace.Create).Methods("GET")
	auth.HandleFunc("/namespaces", namespace.Store).Methods("POST")
	auth.HandleFunc("/settings/invites", namespace.Invite.Index).Methods("GET")
	auth.HandleFunc("/invites/{invite:[0-9]+}", namespace.Collaborator.Store).Methods("PATCH")
	auth.HandleFunc("/invites/{invite:[0-9]+}", namespace.Invite.Destroy).Methods("DELETE")
	auth.Use(r.Middleware.Auth, csrf)

	sr := mux.PathPrefix("/n/{username}/{namespace:[a-zA-Z0-9\\/?]+}").Subrouter()
	sr.HandleFunc("", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/edit", namespace.Edit).Methods("GET")
	sr.HandleFunc("/-/namespaces", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/images", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/objects", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/variables", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/keys", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/collaborators", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/collaborators", namespace.Invite.Store).Methods("POST")
	sr.HandleFunc("/-/collaborators/{collaborator}", namespace.Collaborator.Destroy).Methods("DELETE")
	sr.HandleFunc("", namespace.Update).Methods("PATCH")
	sr.HandleFunc("", namespace.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...), csrf)
}

// RegisterAPI registers the API routes for namespace creation and management.
// There are two types of routes, simple auth routes, and individual namespace
// routes. These routes respond with a application/json Content-Type.
//
// simple auth routes - These routes (/namespaces, /namespaces/create,
// /settings/invites, /invites/{invite:[0-9]+}) havbe the auth middleware
// applied to them to check if a user is authenticated to access the route.
//
// individual namespace routes - These routes (prefixed with
// /n/{username}/{namespace:[a-zA-Z0-9\\/?]+}), use the given gates for auth
// checks, and permission checks.
func (r *Router) RegisterAPI(prefix string, mux *mux.Router, gates ...web.Gate) {
}
