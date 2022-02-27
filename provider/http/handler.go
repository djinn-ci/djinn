package http

import (
	"net/http"
	"strconv"

	"djinn-ci.com/errors"
	"djinn-ci.com/provider"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
)

type Handler struct {
	*server.Server

	Providers *provider.Store
	Repos     provider.RepoStore
	Users     *user.Store
}

type HandlerFunc func(*user.User, *provider.Repo, http.ResponseWriter, *http.Request)

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server: srv,
		Providers: &provider.Store{
			Pool:    srv.DB,
			AESGCM:  srv.AESGCM,
			Clients: srv.Providers,
			Cache:   provider.NewRepoCache(srv.Redis),
		},
		Repos: provider.RepoStore{Pool: srv.DB},
		Users: &user.Store{Pool: srv.DB},
	}
}

func (h *Handler) WithRepo(fn HandlerFunc) userhttp.HandlerFunc {
	return func(u *user.User, w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(mux.Vars(r)["repo"], 10, 64)

		repo, ok, err := h.Repos.Get(query.Where("id", "=", query.Arg(id)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}
		fn(u, repo, w, r)
	}
}
