package http

import (
	"io"
	"net/http"

	"djinn-ci.com/auth"
	"djinn-ci.com/errors"
	"djinn-ci.com/image"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"

	"github.com/andrewpillar/fs"
	"github.com/andrewpillar/webutil/v2"
)

type API struct {
	*Handler
}

func (h API) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	p, err := h.Handler.Index(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get images"))
		return
	}

	w.Header().Set("Link", p.EncodeToLink(r.URL))
	webutil.JSON(w, p.Items, http.StatusOK)
}

func (h API) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	i, _, err := h.Handler.Store(u, r)

	if err != nil {
		h.FormError(w, r, nil, err)
		return
	}
	webutil.JSON(w, i, http.StatusCreated)
}

func (h API) Show(u *auth.User, i *image.Image, w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Accept") == image.MimeTypeQEMU {
		f, err := i.Open(h.Images)

		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				h.NotFound(w, r)
				return
			}

			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get image"))
			return
		}

		defer f.Close()

		w.Header().Set("Content-Type", image.MimeTypeQEMU)
		http.ServeContent(w, r, i.Name, i.CreatedAt, f.(io.ReadSeeker))
		return
	}

	if err := image.LoadRelations(r.Context(), h.DB, i); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to load relations"))
		return
	}
	webutil.JSON(w, i, http.StatusOK)
}

func (h API) Destroy(u *auth.User, i *image.Image, w http.ResponseWriter, r *http.Request) {
	if err := h.Handler.Destroy(r.Context(), i); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to delete image"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func RegisterAPI(a auth.Authenticator, srv *server.Server) {
	api := API{
		Handler: NewHandler(srv),
	}

	index := api.Restrict(a, []string{"image:read"}, api.Index)
	store := api.Restrict(a, []string{"image:write"}, api.Store)

	a = namespace.NewAuth[*image.Image](a, "image", api.Images.Store)

	show := api.Optional(a, api.Image(api.Show))
	destroy := api.Restrict(a, []string{"image:write"}, api.Image(api.Destroy))

	sr := srv.Router.PathPrefix("/images").Subrouter()
	sr.HandleFunc("", index).Methods("GET")
	sr.HandleFunc("", store).Methods("POST")
	sr.HandleFunc("/{image:[0-9]+}", show).Methods("GET")
	sr.HandleFunc("/{image:[0-9]+}", destroy).Methods("DELETE")
}
