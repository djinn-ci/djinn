package web

import (
	"context"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/repo"
	"github.com/andrewpillar/thrall/repo/handler"
	"github.com/andrewpillar/thrall/server"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"
)

type Router struct {
	repo handler.Repo

	Redis      *redis.Client
	Providers  map[string]oauth2.Provider
	Middleware web.Middleware
}

var _ server.Router = (*Router)(nil)

func Gate(db *sqlx.DB) web.Gate {
	repos := repo.NewStore(db)

	onlyAuth := map[string]struct{}{
		"repos":  {},
		"reload": {},
		"enable": {},
	}

	return func(u *user.User, r *http.Request) (*http.Request, bool, error) {
		base := filepath.Base(r.URL.Path)

		if _, ok := onlyAuth[base]; ok {
			return r, !u.IsZero(), nil
		}

		id, _ := strconv.ParseInt(mux.Vars(r)["repo"], 10, 64)

		rp, err := repos.Get(query.Where("id", "=", id))

		if err != nil {
			return r, false, errors.Err(err)
		}

		if rp.IsZero() {
			return r, false, nil
		}

		r = r.WithContext(context.WithValue(r.Context(), "repo", rp))
		return r, rp.UserID == u.ID, nil
	}
}

func (r *Router) Init(h web.Handler) {
	r.repo = handler.Repo{
		Handler:   h,
		Redis:     r.Redis,
		Providers: r.Providers,
		Repos:     repo.NewStore(h.DB),
	}
}

func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/repos", r.repo.Index).Methods("GET")
	auth.HandleFunc("/repos/reload", r.repo.Update).Methods("PATCH")
	auth.HandleFunc("/repos/enable", r.repo.Store).Methods("POST")
	auth.HandleFunc("/repos/disable/{repo:[0-9]+}", r.repo.Destroy).Methods("DELETE")
	auth.Use(r.Middleware.Gate(gates...), csrf)
}

func (r *Router) RegisterAPI(_ string, _ *mux.Router, _ ...web.Gate) {}
