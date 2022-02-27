package http

import (
	"net/http"

	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/webutil"
)

type API struct {
	*Handler

	Prefix string
}

func (h API) Index(u *user.User, w http.ResponseWriter, r *http.Request) {
	vv, paginator, err := h.IndexWithRelations(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	data := make([]map[string]interface{}, 0, len(vv))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, v := range vv {
		data = append(data, v.JSON(addr))
	}

	w.Header().Set("Link", paginator.EncodeToLink(r.URL))
	webutil.JSON(w, data, http.StatusOK)
}

func (h API) Store(u *user.User, w http.ResponseWriter, r *http.Request) {
	v, _, err := h.StoreModel(u, r)

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
	webutil.JSON(w, v.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

func (h API) Show(u *user.User, v *variable.Variable, w http.ResponseWriter, r *http.Request) {
	if err := variable.LoadRelations(h.DB, v); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if err := variable.LoadNamespaces(h.DB, v); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	webutil.JSON(w, v.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

func (h API) Destroy(u *user.User, v *variable.Variable, w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r.Context(), v); err != nil {
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

	sr := srv.Router.PathPrefix("/variables").Subrouter()
	sr.HandleFunc("", user.WithUser(api.Index)).Methods("GET")
	sr.HandleFunc("", user.WithUser(api.Store)).Methods("POST")
	sr.HandleFunc("/{variable:[0-9]+}", user.WithUser(api.WithVariable(api.Show))).Methods("GET")
	sr.HandleFunc("/{variable:[0-9]+}", user.WithUser(api.WithVariable(api.Destroy))).Methods("DELETE")
}
