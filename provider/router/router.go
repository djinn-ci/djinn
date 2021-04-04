package router

import (
	"context"
	"encoding/gob"
	"net/http"
	"strconv"

	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/provider"
	"github.com/andrewpillar/djinn/provider/handler"
	"github.com/andrewpillar/djinn/server"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

// Router is what registers the UI routes for handling integrating with an
// external provider.
type Router struct {
	middleware web.Middleware
	provider   handler.Provider
	repo       handler.Repo
}

var _ server.Router = (*Router)(nil)

// Gate returns a web.Gate that checks if the current authenticated user has
// the access permissions to the provider's repository being accessed. If the
// current user has access ot the current repository, then it is set in the
// request's context.
func Gate(db *sqlx.DB) web.Gate {
	repos := provider.NewRepoStore(db)

	onlyAuth := map[string]struct{}{
		"repos":  {},
		"reload": {},
		"enable": {},
	}

	return func(u *user.User, r *http.Request) (*http.Request, bool, error) {
		base := webutil.BasePath(r.URL.Path)

		if _, ok := onlyAuth[base]; ok {
			return r, !u.IsZero(), nil
		}

		id, _ := strconv.ParseInt(mux.Vars(r)["repo"], 10, 64)

		repo, err := repos.Get(query.Where("id", "=", query.Arg(id)))

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

func New(cfg *config.Server, h web.Handler, mw web.Middleware) *Router {
	gob.Register([]*provider.Repo{})
	gob.Register(database.Paginator{})

	redis := cfg.Redis()
	block := cfg.BlockCipher()
	providers := cfg.Providers()

	return &Router{
		middleware: mw,
		provider:   handler.New(h, redis, block, providers),
		repo:       handler.NewRepo(h, cfg.Redis(), block, providers),
	}
}

// RegisterUI registers the UI routes for handling integration with a provider.
// This will register the routes for connecting to a provider, and the routes
// for toggling webhooks on a provider's repository.
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
	sr.Use(r.middleware.Gate(gates...), csrf)
}

// RegisterAPI is a stub method to implement the server.Router interface.
func (*Router) RegisterAPI(_ string, _ *mux.Router, _ ...web.Gate) {}
