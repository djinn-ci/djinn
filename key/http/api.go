package http

import (
	"net/http"

	"djinn-ci.com/errors"
	"djinn-ci.com/key"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"

	"github.com/andrewpillar/webutil"
)

type API struct {
	*Handler

	Prefix string
}

func (h API) Index(u *user.User, w http.ResponseWriter, r *http.Request) {
	kk, paginator, err := h.IndexWithRelations(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	data := make([]map[string]interface{}, 0, len(kk))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, k := range kk {
		data = append(data, k.JSON(addr))
	}

	w.Header().Set("Link", paginator.EncodeToLink(r.URL))
	webutil.JSON(w, data, http.StatusOK)
}

func (h API) Store(u *user.User, w http.ResponseWriter, r *http.Request) {
	k, _, err := h.StoreModel(u, r)

	if err != nil {
		cause := errors.Cause(err)

		switch err := cause.(type) {
		case webutil.ValidationErrors:
			if errs, ok := err["fatal"]; ok {
				h.InternalServerError(w, r, errors.Slice(errs))
				return
			}
			webutil.JSON(w, err, http.StatusBadRequest)
		case *namespace.PathError:
			webutil.JSON(w, map[string][]string{"namespace": {err.Error()}}, http.StatusBadRequest)
		default:
			h.InternalServerError(w, r, errors.Err(err))
		}
		return
	}
	webutil.JSON(w, k.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

func (h API) Show(u *user.User, k *key.Key, w http.ResponseWriter, r *http.Request) {
	if err := key.LoadRelations(h.DB, k); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if err := key.LoadNamespaces(h.DB, k); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	webutil.JSON(w, k.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

func (h API) Update(u *user.User, k *key.Key, w http.ResponseWriter, r *http.Request) {
	k, _, err := h.UpdateModel(k, r)

	if err != nil {
		cause := errors.Cause(err)

		if verrs, ok := cause.(webutil.ValidationErrors); ok {
			if errs, ok := verrs["fatal"]; ok {
				h.InternalServerError(w, r, errors.Slice(errs))
				return
			}

			webutil.JSON(w, verrs, http.StatusBadRequest)
			return
		}

		if errors.Is(err, namespace.ErrPermission) {
			webutil.JSON(w, map[string][]string{"namespace": {"Could not find namespace"}}, http.StatusBadRequest)
			return
		}

		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	webutil.JSON(w, k.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

func (h API) Destroy(u *user.User, k *key.Key, w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r.Context(), k); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func RegisterAPI(prefix string, srv *server.Server) {
	user := userhttp.NewHandler(srv)

	api := API{
		Handler: NewHandler(srv),
		Prefix:  prefix,
	}

	sr := srv.Router.PathPrefix("/keys").Subrouter()
	sr.HandleFunc("", user.WithUser(api.Index)).Methods("GET")
	sr.HandleFunc("", user.WithUser(api.Store)).Methods("POST")
	sr.HandleFunc("/{key:[0-9]+}", user.WithUser(api.WithKey(api.Show))).Methods("GET")
	sr.HandleFunc("/{key:[0-9]+}", user.WithUser(api.WithKey(api.Update))).Methods("PATCH")
	sr.HandleFunc("/{key:[0-9]+}", user.WithUser(api.WithKey(api.Destroy))).Methods("DELETE")
}
