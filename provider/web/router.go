package web

import (
	"context"
	"encoding/gob"
	"net/http"
	"strconv"

	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/provider"
	"github.com/andrewpillar/djinn/provider/handler"
	"github.com/andrewpillar/djinn/server"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"
)

type Router struct {
	provider handler.Provider
	repo     handler.Repo

	Redis      *redis.Client
	Block      *crypto.Block
	Registry   *provider.Registry
	Middleware web.Middleware
}

var _ server.Router = (*Router)(nil)

func Gate(db *sqlx.DB) web.Gate {
	repos := provider.NewRepoStore(db)

	onlyAuth := map[string]struct{}{
		"repos":  {},
		"reload": {},
		"enable": {},
	}

	return func(u *user.User, r *http.Request) (*http.Request, bool, error) {
		base := web.BasePath(r.URL.Path)

		if _, ok := onlyAuth[base]; ok {
			return r, !u.IsZero(), nil
		}

		id, _ := strconv.ParseInt(mux.Vars(r)["repo"], 10, 64)

		repo, err := repos.Get(query.Where("id", "=", id))

		if err != nil {
			return r, false, errors.Err(err)
		}

		if repo.IsZero() {
			return r, false, nil
		}

		r = r.WithContext(context.WithValue(r.Context(), "repo", repo))
		return r, repo.UserID == u.ID, nil
	}
}

func (r *Router) Init(h web.Handler) {
	gob.Register([]*provider.Repo{})
	gob.Register(database.Paginator{})

	r.provider = handler.Provider{
		Handler:  h,
		Block:    r.Block,
		Registry: r.Registry,
	}
	r.repo = handler.Repo{
		Handler:  h,
		Redis:    r.Redis,
		Block:    r.Block,
		Registry: r.Registry,
		Repos:    provider.NewRepoStore(h.DB),
	}
}

func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	auth := mux.PathPrefix("/oauth").Subrouter()
	auth.HandleFunc("/{provider}", r.provider.Auth).Methods("GET")
	auth.HandleFunc("/{provider}", r.provider.Revoke).Methods("DELETE")
	auth.Use(csrf)

	sr := mux.PathPrefix("/repos").Subrouter()
	sr.HandleFunc("", r.repo.Index).Methods("GET")
	sr.HandleFunc("/reload", r.repo.Update).Methods("PATCH")
	sr.HandleFunc("/enable", r.repo.Store).Methods("POST")
	sr.HandleFunc("/disable/{repo:[0-9]+}", r.repo.Destroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...), csrf)
}

func (*Router) RegisterAPI(_ string, _ *mux.Router, _ ...web.Gate) {}
