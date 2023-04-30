package http

import (
	"net/http"

	"djinn-ci.com/auth"
	"djinn-ci.com/build"
	"djinn-ci.com/cron"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"
)

type API struct {
	*Handler
}

func (h API) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	p, err := h.Handler.Index(u, r)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get cron jobs"))
		return
	}

	w.Header().Set("Link", p.EncodeToLink(r.URL))
	webutil.JSON(w, p.Items, http.StatusOK)
}

func (h API) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	c, _, err := h.Handler.Store(u, r)

	if err != nil {
		h.FormError(w, r, nil, err)
		return
	}
	webutil.JSON(w, c, http.StatusCreated)
}

func (h API) Show(u *auth.User, c *cron.Cron, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if webutil.BasePath(r.URL.Path) == "builds" {
		if !u.Has("build:read") {
			h.NotFound(w, r)
			return
		}

		p, err := h.Builds.Index(ctx, r.URL.Query(), query.Where("id", "IN", cron.SelectBuild(
			query.Columns("build_id"),
			query.Where("cron_id", "=", query.Arg(c.ID)),
		)))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get builds"))
			return
		}

		if err := build.LoadRelations(ctx, h.DB, p.Items...); err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to load build relations"))
			return
		}

		w.Header().Set("Link", p.EncodeToLink(r.URL))
		webutil.JSON(w, p.Items, http.StatusOK)
		return
	}
	webutil.JSON(w, c, http.StatusOK)
}

func (h API) Update(u *auth.User, c *cron.Cron, w http.ResponseWriter, r *http.Request) {
	c, _, err := h.Handler.Update(u, c, r)

	if err != nil {
		h.FormError(w, r, nil, err)
		return
	}
	webutil.JSON(w, c, http.StatusOK)
}

func (h API) Destroy(u *auth.User, c *cron.Cron, w http.ResponseWriter, r *http.Request) {
	if err := h.Handler.Destroy(r.Context(), c); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to delete cron job"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func RegisterAPI(a auth.Authenticator, srv *server.Server) {
	api := API{
		Handler: NewHandler(srv),
	}

	index := api.Restrict(a, []string{"cron:read"}, api.Index)
	store := api.Restrict(a, []string{"cron:write"}, api.Store)

	a = namespace.NewAuth(a, "cron", cron.NewStore(srv.DB))

	show := api.Restrict(a, []string{"cron:read"}, api.Cron(api.Show))
	update := api.Restrict(a, []string{"cron:write"}, api.Cron(api.Update))
	destroy := api.Restrict(a, []string{"cron:delete"}, api.Cron(api.Destroy))

	sr := srv.Router.PathPrefix("/cron").Subrouter()
	sr.HandleFunc("", index).Methods("GET")
	sr.HandleFunc("", store).Methods("POST")
	sr.HandleFunc("/{cron:[0-9]+}", show).Methods("GET")
	sr.HandleFunc("/{cron:[0-9]+}/builds", show).Methods("GET")
	sr.HandleFunc("/{cron:[0-9]+}", update).Methods("PATCH")
	sr.HandleFunc("/{cron:[0-9]+}", destroy).Methods("DELETE")
}
