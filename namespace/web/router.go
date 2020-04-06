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

func Gate(db *sqlx.DB) web.Gate {
	users := user.NewStore(db)
	namespaces := namespace.NewStore(db)

	return func(u *user.User, r *http.Request) (*http.Request, bool, error) {
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

		r = r.WithContext(context.WithValue(r.Context(), "namespace", n))

		if filepath.Base(r.URL.Path) == "edit" {
			return r, u.ID == n.UserID, nil
		}

		if r.Method == "DELETE" || r.Method == "PATCH" {
			return r, u.ID == n.UserID, nil
		}

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

func (r *Router) Init(h web.Handler) {
	namespaces := namespace.NewStore(h.DB)

	loaders := model.NewLoaders()
	loaders.Put("namespace", namespaces)
	loaders.Put("user", h.Users)
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
		Namespaces: namespaces,
		Invites:    namespace.NewInviteStore(h.DB),
	}
}

func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	namespace := handler.UI{
		Namespace:    r.namespace,
		Invite:       handler.InviteUI{
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
	auth.HandleFunc("/settings/invites/{invite:[0-9]+}", namespace.Collaborator.Store).Methods("PATCH")
	auth.HandleFunc("/settings/invites/{invite:[0-9]+}", namespace.Collaborator.Destroy).Methods("DELETE")
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

func (r *Router) RegisterAPI(mux *mux.Router, gates ...web.Gate) {}
