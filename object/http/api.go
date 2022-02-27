package http

import (
	"net/http"

	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/namespace"
	"djinn-ci.com/object"
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
	oo, paginator, err := h.IndexWithRelations(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	data := make([]map[string]interface{}, 0, len(oo))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, o := range oo {
		data = append(data, o.JSON(addr))
	}

	w.Header().Set("Link", paginator.EncodeToLink(r.URL))
	webutil.JSON(w, data, http.StatusOK)
}

func (h API) Store(u *user.User, w http.ResponseWriter, r *http.Request) {
	o, _, err := h.StoreModel(u, r)

	if err != nil {
		cause := errors.Cause(err)

		if verrs, ok := cause.(webutil.ValidationErrors); ok {
			webutil.JSON(w, verrs, http.StatusBadRequest)
			return
		}

		errs := webutil.NewValidationErrors()

		switch cause {
		case namespace.ErrName:
			errs.Add("namespace", cause)

			webutil.JSON(w, errs, http.StatusBadRequest)
		case namespace.ErrPermission, namespace.ErrOwner:
			webutil.JSON(w, map[string][]string{"namespace": {"Could not find namespace"}}, http.StatusBadRequest)
		default:
			h.InternalServerError(w, r, errors.Err(err))
		}
		return
	}
	webutil.JSON(w, o.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

func (h API) Show(u *user.User, o *object.Object, w http.ResponseWriter, r *http.Request) {
	if err := object.LoadRelations(h.DB, o); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if err := object.LoadNamespaces(h.DB, o); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	addr := webutil.BaseAddress(r) + h.Prefix
	base := webutil.BasePath(r.URL.Path)

	if base == "builds" {
		bb, paginator, err := h.getBuilds(o, r)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		data := make([]map[string]interface{}, 0, len(bb))

		for _, b := range bb {
			data = append(data, b.JSON(addr))
		}

		w.Header().Set("Link", paginator.EncodeToLink(r.URL))
		webutil.JSON(w, data, http.StatusOK)
		return
	}

	if r.Header.Get("Accept") == o.Type {
		rec, err := o.Data(h.Objects.Store)

		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				h.NotFound(w, r)
				return
			}

			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		defer rec.Close()

		http.ServeContent(w, r, o.Name, o.CreatedAt, rec)
		return
	}
	webutil.JSON(w, o.JSON(addr), http.StatusOK)
}

func (h API) Destroy(u *user.User, o *object.Object, w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r.Context(), o); err != nil {
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

	sr := srv.Router.PathPrefix("/objects").Subrouter()
	sr.HandleFunc("", user.WithUser(api.Index)).Methods("GET")
	sr.HandleFunc("", user.WithUser(api.Store)).Methods("POST")
	sr.HandleFunc("/{object:[0-9]+}", user.WithUser(api.WithObject(api.Show))).Methods("GET")
	sr.HandleFunc("/{object:[0-9]+}/builds", user.WithUser(api.WithObject(api.Show))).Methods("GET")
	sr.HandleFunc("/{object:[0-9]+}", user.WithUser(api.WithObject(api.Destroy))).Methods("DELETE")
}
