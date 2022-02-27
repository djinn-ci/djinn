package http

import (
	"net/http"
	"os"

	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/image"
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
	ii, paginator, err := h.IndexWithRelations(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	data := make([]map[string]interface{}, 0, len(ii))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, i := range ii {
		data = append(data, i.JSON(addr))
	}

	w.Header().Set("Link", paginator.EncodeToLink(r.URL))
	webutil.JSON(w, data, http.StatusOK)
}

func (h API) Store(u *user.User, w http.ResponseWriter, r *http.Request) {
	i, _, err := h.StoreModel(u, r)

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

		if _, ok := cause.(*os.PathError); ok {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		errs := webutil.NewValidationErrors()

		switch cause {
		case image.ErrInvalidScheme:
			errs.Add("download_url", cause)

			webutil.JSON(w, errs, http.StatusBadRequest)
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
	webutil.JSON(w, i.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

func (h API) Show(u *user.User, i *image.Image, w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Accept") == image.MimeTypeQEMU {
		rec, err := i.Data(h.Images.Store)

		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				h.NotFound(w, r)
				return
			}

			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		defer rec.Close()

		w.Header().Set("Content-Type", image.MimeTypeQEMU)
		http.ServeContent(w, r, i.Name, i.CreatedAt, rec)
		return
	}

	if err := image.LoadRelations(h.DB, i); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if err := image.LoadNamespaces(h.DB, i); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	webutil.JSON(w, i.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

func (h API) Destroy(u *user.User, i *image.Image, w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r.Context(), i); err != nil {
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

	sr := srv.Router.PathPrefix("/images").Subrouter()
	sr.HandleFunc("", user.WithUser(api.Index)).Methods("GET")
	sr.HandleFunc("", user.WithUser(api.Store)).Methods("POST")
	sr.HandleFunc("/{image:[0-9]+}", user.WithUser(api.WithImage(api.Show))).Methods("GET")
	sr.HandleFunc("/{image:[0-9]+}", user.WithUser(api.WithImage(api.Destroy))).Methods("DELETE")
}
