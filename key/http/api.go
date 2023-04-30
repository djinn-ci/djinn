package http

import (
	"net/http"

	"djinn-ci.com/auth"
	"djinn-ci.com/errors"
	"djinn-ci.com/key"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"

	"github.com/andrewpillar/webutil/v2"
)

type API struct {
	*Handler
}

func (h API) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	p, err := h.Handler.Index(u, r)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get keys"))
		return
	}

	w.Header().Set("Link", p.EncodeToLink(r.URL))
	webutil.JSON(w, p.Items, http.StatusOK)
}

func (h API) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	k, _, err := h.Handler.Store(u, r)

	if err != nil {
		h.FormError(w, r, nil, err)
		return
	}
	webutil.JSON(w, k, http.StatusCreated)
}

func (h API) Show(u *auth.User, k *key.Key, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := namespace.LoadResourceRelations[*key.Key](ctx, h.DB, k); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to load relations"))
		return
	}
	webutil.JSON(w, k, http.StatusOK)
}

func (h API) Update(u *auth.User, k *key.Key, w http.ResponseWriter, r *http.Request) {
	k, _, err := h.Handler.Update(u, k, r)

	if err != nil {
		h.FormError(w, r, nil, err)
		return
	}
	webutil.JSON(w, k, http.StatusOK)
}

func (h API) Destroy(u *auth.User, k *key.Key, w http.ResponseWriter, r *http.Request) {
	if err := h.Handler.Destroy(r.Context(), k); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to delete key"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func RegisterAPI(a auth.Authenticator, srv *server.Server) {
	api := API{
		Handler: NewHandler(srv),
	}

	index := api.Restrict(a, []string{"key:read"}, api.Index)
	store := api.Restrict(a, []string{"key:write"}, api.Store)

	a = namespace.NewAuth(a, "key", api.Keys.Store)

	show := api.Restrict(a, []string{"key:read"}, api.Key(api.Show))
	update := api.Restrict(a, []string{"key:write"}, api.Key(api.Update))
	destroy := api.Restrict(a, []string{"key:delete"}, api.Key(api.Destroy))

	sr := srv.Router.PathPrefix("/keys").Subrouter()
	sr.HandleFunc("", index).Methods("GET")
	sr.HandleFunc("", store).Methods("POST")
	sr.HandleFunc("/{key:[0-9]+}", show).Methods("GET")
	sr.HandleFunc("/{key:[0-9]+}", update).Methods("PATCH")
	sr.HandleFunc("/{key:[0-9]+}", destroy).Methods("DELETE")
}
