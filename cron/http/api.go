package http

import (
	"net/http"

	"djinn-ci.com/cron"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

type API struct {
	*Handler

	Prefix string
}

func (h API) Index(u *user.User, w http.ResponseWriter, r *http.Request) {
	cc, paginator, err := h.IndexWithRelations(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	data := make([]map[string]interface{}, 0, len(cc))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, c := range cc {
		data = append(data, c.JSON(addr))
	}

	w.Header().Set("Link", paginator.EncodeToLink(r.URL))
	webutil.JSON(w, data, http.StatusOK)
}

func (h API) Store(u *user.User, w http.ResponseWriter, r *http.Request) {
	c, _, err := h.StoreModel(u, r)

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
	webutil.JSON(w, c.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

func (h API) Show(u *user.User, c *cron.Cron, w http.ResponseWriter, r *http.Request) {
	if err := cron.LoadRelations(h.DB, c); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if err := namespace.Load(h.DB, c); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	addr := webutil.BaseAddress(r) + h.Prefix

	if webutil.BasePath(r.URL.Path) == "builds" {
		bb, paginator, err := h.Builds.Index(r.URL.Query(), query.Where("id", "IN", cron.SelectBuildIDs(c.ID)))

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
	webutil.JSON(w, c.JSON(addr), http.StatusOK)
}

func (h API) Update(u *user.User, c *cron.Cron, w http.ResponseWriter, r *http.Request) {
	c, _, err := h.UpdateModel(c, r)

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

		switch cause {
		case namespace.ErrName:
			errs := webutil.NewValidationErrors()
			errs.Add("namespace", cause)

			webutil.JSON(w, errs, http.StatusBadRequest)
		case namespace.ErrPermission, namespace.ErrOwner:
			webutil.JSON(w, map[string][]string{"namespace": {"Could not find namespace"}}, http.StatusBadRequest)
		default:
			h.InternalServerError(w, r, errors.Err(err))
		}
		return
	}
	webutil.JSON(w, c.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

func (h API) Destroy(u *user.User, c *cron.Cron, w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r.Context(), c); err != nil {
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

	sr := srv.Router.PathPrefix("/cron").Subrouter()
	sr.HandleFunc("", user.WithUser(api.Index)).Methods("GET")
	sr.HandleFunc("", user.WithUser(api.Store)).Methods("POST")
	sr.HandleFunc("/{cron:[0-9]+}", user.WithUser(api.WithCron(api.Show))).Methods("GET")
	sr.HandleFunc("/{cron:[0-9]+}/builds", user.WithUser(api.WithCron(api.Show))).Methods("GET")
	sr.HandleFunc("/{cron:[0-9]+}", user.WithUser(api.WithCron(api.Update))).Methods("PATCH")
	sr.HandleFunc("/{cron:[0-9]+}", user.WithUser(api.WithCron(api.Destroy))).Methods("DELETE")
}
