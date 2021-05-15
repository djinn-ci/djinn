package router

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"djinn-ci.com/config"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/namespace/handler"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	"djinn-ci.com/web"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

// Router is what registers the UI and API routes for managing namespaces. It
// implements the server.Router interface.
type Router struct {
	middleware   web.Middleware
	namespace    handler.Namespace
	invite       handler.Invite
	webhook      handler.Webhook
	collaborator handler.Collaborator
}

var _ server.Router = (*Router)(nil)

// Gate returns a web.Gate that checks if the current authenticated User has
// the access permissions to the current Namespace, or if they are a
// Collaborator in that Namespace. If the current User can access the current
// Namespace, then it is set in the request's context.
func Gate(db *sqlx.DB) web.Gate {
	users := user.NewStore(db)
	namespaces := namespace.NewStore(db)

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

		base := webutil.BasePath(r.URL.Path)

		if base == "create" {
			parts := strings.Split(r.URL.Path, "/")

			// Not creating a webhook so return. If creating a webhook we want
			// to get the namespace and owner from the URL.
			if parts[len(parts)-2] != "webhooks" {
				return r, ok, nil
			}
		}

		// Check if the base of the path is for a namespace's children or
		// invites. This is denoted by a - preceding the base of the path.
		if base == "namespaces" || base == "invites" {
			parts := strings.Split(r.URL.Path, "/")

			if parts[len(parts)-2] != "-" {
				return r, ok, nil
			}
		}

		vars := mux.Vars(r)

		// Invites are a subresource of the namespace entity. So we want to do
		// the gate checks here if the user is creating or modifying an invite.
		if invite, ok := vars["invite"]; ok {
			id, _ := strconv.ParseInt(invite, 10, 64)

			switch r.Method {
			case "GET":
				_, ok = u.Permissions["invite:read"]
			case "POST", "PATCH":
				_, ok = u.Permissions["invite:write"]
			case "DELETE":
				_, ok = u.Permissions["invite:delete"]
			}

			i, err := namespace.NewInviteStore(db).Get(query.Where("id", "=", query.Arg(id)))

			if err != nil {
				return r, ok, errors.Err(err)
			}

			if i.IsZero() {
				return r, false, nil
			}

			r = r.WithContext(context.WithValue(r.Context(), "invite", i))
			return r, ok, errors.Err(err)
		}

		if webhook, ok := vars["webhook"]; ok {
			id, _ := strconv.ParseInt(webhook, 10, 64)

			switch r.Method {
			case "GET":
				_, ok = u.Permissions["webhook:read"]
			case "PATCH":
				_, ok = u.Permissions["webhook:write"]
			case "DELETE":
				_, ok = u.Permissions["webhook:delete"]
			}

			w, err := namespace.NewWebhookStore(db).Get(query.Where("id", "=", query.Arg(id)))

			if err != nil {
				return r, ok, errors.Err(err)
			}

			if w.IsZero() {
				return r, false, nil
			}

			r = r.WithContext(context.WithValue(r.Context(), "webhook", w))
			return r, ok, errors.Err(err)
		}

		owner, err := users.Get(query.Where("username", "=", query.Arg(vars["username"])))

		if err != nil {
			return r, false, errors.Err(err)
		}

		if owner.IsZero() {
			return r, false, nil
		}

		path := strings.TrimSuffix(vars["namespace"], "/")

		n, err := namespace.NewStore(db, owner).Get(
			database.Where(owner, "user_id"),
			query.Where("path", "=", query.Arg(path)),
		)

		if err != nil {
			return r, false, errors.Err(err)
		}

		if n.IsZero() {
			return r, false, errors.Err(err)
		}

		if n.UserID != owner.ID {
			return r, false, nil
		}

		// Can the current user modify/delete the current namespace.
		if r.Method == "POST" || r.Method == "PATCH" || r.Method == "DELETE" {
			if owner.ID != u.ID {
				return r, false, nil
			}
		}

		// Can the current user view the namespace invites/collaborators, or
		// edit the namespace.
		if base == "invites" || base == "collaborators" || base == "edit" {
			if owner.ID != u.ID {
				return r, false, nil
			}
		}

		r = r.WithContext(context.WithValue(r.Context(), "namespace", n))

		root, err := namespaces.Get(
			query.Where("root_id", "=", namespace.SelectRootID(n.ID)),
			query.Where("id", "=", namespace.SelectRootID(n.ID)),
		)

		if err != nil {
			return r, false, errors.Err(err)
		}

		cc, err := namespace.NewCollaboratorStore(db, root).All()

		if err != nil {
			return r, false, errors.Err(err)
		}

		root.LoadCollaborators(cc)
		return r, ok && root.AccessibleBy(u), nil
	}
}

func New(_ *config.Server, h web.Handler, mw web.Middleware) *Router {
	return &Router{
		middleware:   mw,
		namespace:    handler.New(h),
		invite:       handler.NewInvite(h),
		webhook:      handler.NewWebhook(h),
		collaborator: handler.Collaborator{Handler: h},
	}
}

// RegisterUI registers the UI routes for Namespace creation, and management.
// There are two types of routes, simple auth routes, and individual namespace
// routes. These routes respond with a text/html Content-Type.
//
// simple auth routes - These routes (/namespaces, /namespaces/create,
// /invites, /invites/{invite:[0-9]+}) havbe the auth middleware applied to
// them to check if a user is authenticated to access the route. The given
// http.Handler is applied to these routes for CSRF protection.
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

	webhook := handler.WebhookUI{
		Webhook: r.webhook,
	}

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/namespaces", namespace.Index).Methods("GET")
	auth.HandleFunc("/namespaces/create", namespace.Create).Methods("GET")
	auth.HandleFunc("/namespaces", namespace.Store).Methods("POST")
	auth.HandleFunc("/invites", invite.Index).Methods("GET")
	auth.HandleFunc("/invites/{invite:[0-9]+}", invite.Update).Methods("PATCH")
	auth.HandleFunc("/invites/{invite:[0-9]+}", invite.Destroy).Methods("DELETE")
	auth.Use(r.middleware.Auth, r.middleware.Gate(gates...), csrf)

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
	sr.HandleFunc("/-/webhooks", webhook.Index).Methods("GET")
	sr.HandleFunc("/-/webhooks/create", webhook.Create).Methods("GET")
	sr.HandleFunc("/-/webhooks", webhook.Store).Methods("POST")
	sr.HandleFunc("/-/webhooks/{webhook:[0-9]+}", webhook.Show).Methods("GET")
//	sr.HandleFunc("/-/webhooks/{webhook:[0-9]+}", webhook.Update).Methods("PATCH")
//	sr.HandleFunc("/-/webhooks/{webhook:[0-9]+}/events", webhook.Show).Methods("GET")
	sr.HandleFunc("", namespace.Update).Methods("PATCH")
	sr.HandleFunc("", namespace.Destroy).Methods("DELETE")
	sr.Use(r.middleware.Gate(gates...), csrf)
}

// RegisterAPI registers the API routes for working with namespaces. The given
// prefix string is used to specify where the API is being served under. This
// applies all of the given gates to all routes registered. These routes
// response with a "application/json" Content-Type.
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
	auth.Use(r.middleware.Gate(gates...))

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
	sr.Use(r.middleware.Gate(gates...))
}
