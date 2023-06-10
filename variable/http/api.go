package http

import (
	"net/http"

	"djinn-ci.com/auth"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/webutil/v2"
)

type API struct {
	*Handler
}

func (h API) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	paginator, err := h.Handler.Index(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get variables"))
		return
	}

	w.Header().Set("Link", paginator.EncodeToLink(r.URL))
	webutil.JSON(w, paginator.Items, http.StatusOK)
}

func (h API) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	v, _, err := h.Handler.Store(u, r)

	if err != nil {
		h.FormError(w, r, nil, err)
		return
	}
	webutil.JSON(w, v, http.StatusCreated)
}

func (h API) Show(u *auth.User, v *variable.Variable, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := namespace.LoadResourceRelations[*variable.Variable](ctx, h.DB, v); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to load relations"))
		return
	}
	webutil.JSON(w, v, http.StatusOK)
}

func (h API) Destroy(u *auth.User, v *variable.Variable, w http.ResponseWriter, r *http.Request) {
	if err := h.Handler.Destroy(r.Context(), v); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to delete variable"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func RegisterAPI(a auth.Authenticator, srv *server.Server) {
	api := API{
		Handler: NewHandler(srv),
	}

	index := api.Restrict(a, []string{"variable:read"}, api.Index)
	store := api.Restrict(a, []string{"variable:write"}, api.Store)

	a = namespace.NewAuth[*variable.Variable](a, "variable", api.Variables.Store)

	show := api.Restrict(a, []string{"variable:read"}, api.Variable(api.Show))
	destroy := api.Restrict(a, []string{"variable:delete"}, api.Variable(api.Destroy))

	sr := srv.Router.PathPrefix("/variables").Subrouter()
	sr.HandleFunc("", index).Methods("GET")
	sr.HandleFunc("", store).Methods("POST")
	sr.HandleFunc("/{variable:[0-9]+}", show).Methods("GET")
	sr.HandleFunc("/{variable:[0-9]+}", destroy).Methods("DELETE")
}
