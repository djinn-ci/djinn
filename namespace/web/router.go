package web

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/database"
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

		base := filepath.Base(r.URL.Path)

		if base == "create" {
			return r, ok, nil
		}

		// Check if the base of the path is for a namespace, as denoted by a
		// preceding -, for example,
		//
		//     /n/ford/cradle/-/namespaces
		//     /n/ford/cradle/-/invites
		//
		// If not, then return early with the information we have about the
		// user's perms.
		if base == "namespaces" || base == "invites" {
			parts := strings.Split(r.URL.Path, "/")

			if parts[len(parts) - 2] != "-" {
				return r, ok, nil
			}
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
// logic of Namespace creation and management. This will setup the database.Loader
// for relationship loading, and the related database stores. The exported
// properties on the Router itself are pased through to the underlying
// handler.Namspace.
func (r *Router) Init(h web.Handler) {
	namespaces := namespace.NewStore(h.DB)

	loaders := database.NewLoaders()
	loaders.Put("user", h.Users)
	loaders.Put("inviter", h.Users)
	loaders.Put("invitee", h.Users)
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
		Handler:    h,
		Invites:    namespace.NewInviteStore(h.DB),
		Loaders:    loaders,
	}

	r.collaborator = handler.Collaborator{
		Handler: h,
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
	}

	invite := handler.InviteUI{
		Invite: r.invite,
	}

	collaborator := handler.CollaboratorUI{
		Collaborator: r.collaborator,
	}

	perms := []string{
		"namespace:read",
		"namespace:write",
		"invite:read",
		"invite:write",
		"invite:delete",
	}

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/namespaces", namespace.Index).Methods("GET")
	auth.HandleFunc("/namespaces/create", namespace.Create).Methods("GET")
	auth.HandleFunc("/namespaces", namespace.Store).Methods("POST")
	auth.HandleFunc("/settings/invites", invite.Index).Methods("GET")
	auth.HandleFunc("/invites/{invite:[0-9]+}", invite.Update).Methods("PATCH")
	auth.HandleFunc("/invites/{invite:[0-9]+}", invite.Destroy).Methods("DELETE")
	auth.Use(r.Middleware.AuthPerms(perms...), csrf)

	sr := mux.PathPrefix("/n/{username}/{namespace:[a-zA-Z0-9\\/?]+}").Subrouter()
	sr.HandleFunc("", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/edit", namespace.Edit).Methods("GET")
	sr.HandleFunc("/-/namespaces", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/images", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/objects", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/variables", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/keys", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/invites", invite.Index).Methods("GET")
	sr.HandleFunc("/-/invites", invite.Store).Methods("POST")
	sr.HandleFunc("/-/collaborators", collaborator.Index).Methods("GET")
	sr.HandleFunc("/-/collaborators/{collaborator}", collaborator.Destroy).Methods("DELETE")
	sr.HandleFunc("", namespace.Update).Methods("PATCH")
	sr.HandleFunc("", namespace.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...), csrf)
}

// RegisterAPI registers the routes for working with Namespaces over the API.
func (r *Router) RegisterAPI(prefix string, mux *mux.Router, gates ...web.Gate) {
	namespace := handler.API{
		Namespace: r.namespace,
		Prefix:    prefix,
	}

	invite := handler.InviteAPI{
		Invite: r.invite,
		Prefix: prefix,
	}

	collaborator := handler.CollaboratorAPI{
		Collaborator: r.collaborator,
		Prefix:       prefix,
	}

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/namespaces", namespace.Index).Methods("GET", "HEAD")
	auth.HandleFunc("/namespaces", namespace.Store).Methods("POST")
	auth.HandleFunc("/invites", invite.Index).Methods("GET", "HEAD")
	auth.HandleFunc("/invites/{invite:[0-9]+}", invite.Update).Methods("PATCH")
	auth.HandleFunc("/invites/{invite:[0-9]+}", invite.Destroy).Methods("DELETE")
	auth.Use(r.Middleware.Gate(gates...))

	sr := mux.PathPrefix("/n/{username}/{namespace:[a-zA-Z0-9\\/?]+}").Subrouter()
	sr.HandleFunc("", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/builds", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/namespaces", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/images", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/objects", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/variables", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/keys", namespace.Show).Methods("GET")
	sr.HandleFunc("/-/invites", invite.Index).Methods("GET")
	sr.HandleFunc("/-/invites", invite.Store).Methods("POST")
	sr.HandleFunc("/-/collaborators", collaborator.Index).Methods("GET")
	sr.HandleFunc("/-/collaborators/{collaborator}", collaborator.Destroy).Methods("DELETE")
	sr.HandleFunc("", namespace.Update).Methods("PATCH")
	sr.HandleFunc("", namespace.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...))
}
