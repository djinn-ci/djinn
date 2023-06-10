package http

import (
	"net/http"

	"djinn-ci.com/auth"
	"djinn-ci.com/errors"
	"djinn-ci.com/provider"
	"djinn-ci.com/server"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
)

type Handler struct {
	*server.Server

	Providers *provider.Store
	Repos     *provider.RepoStore
}

type HandlerFunc func(*auth.User, *provider.Repo, http.ResponseWriter, *http.Request)

func NewHandler(srv *server.Server) *Handler {
	users := user.Store{
		Store: user.NewStore(srv.DB),
	}

	return &Handler{
		Server: srv,
		Providers: &provider.Store{
			Store:     provider.NewStore(srv.DB),
			AuthStore: users,
			AESGCM:    srv.AESGCM,
			Clients:   srv.Providers,
		},
		Repos: &provider.RepoStore{
			Store: provider.NewRepoStore(srv.DB),
			Cache: srv.Redis,
		},
	}
}

func (h *Handler) Repo(fn HandlerFunc) auth.HandlerFunc {
	return func(u *auth.User, w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["repo"]

		repo, ok, err := h.Repos.Get(r.Context(), query.Where("id", "=", query.Arg(id)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get repo"))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}
		fn(u, repo, w, r)
	}
}
