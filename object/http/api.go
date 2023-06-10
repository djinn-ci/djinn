package http

import (
	"io"
	"net/http"

	"djinn-ci.com/auth"
	"djinn-ci.com/build"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/object"
	"djinn-ci.com/server"

	"github.com/andrewpillar/fs"
	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"
)

type API struct {
	*Handler
}

func (h API) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	p, err := h.Handler.Index(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get objects"))
		return
	}
	w.Header().Set("Link", p.EncodeToLink(r.URL))
	webutil.JSON(w, p.Items, http.StatusOK)
}

func (h API) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	o, _, err := h.Handler.Store(u, r)

	if err != nil {
		h.FormError(w, r, nil, err)
		return
	}
	webutil.JSON(w, o, http.StatusCreated)
}

func (h API) Show(u *auth.User, o *object.Object, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if webutil.BasePath(r.URL.Path) == "builds" {
		if !u.Has("build:read") {
			h.NotFound(w, r)
			return
		}

		p, err := h.Builds.Index(
			ctx,
			r.URL.Query(),
			query.Where("id", "IN", build.SelectObject(
				query.Columns("build_id"),
				query.Where("object_id", "=", query.Arg(o.ID)),
			)),
		)

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get builds"))
			return
		}

		if err := build.LoadRelations(ctx, h.DB, p.Items...); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to load build relations"))
			return
		}

		w.Header().Set("Link", p.EncodeToLink(r.URL))
		webutil.JSON(w, p.Items, http.StatusOK)
		return
	}

	if r.Header.Get("Accept") == o.Type {
		f, err := o.Open(h.Objects)

		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				h.NotFound(w, r)
				return
			}

			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get object"))
			return
		}

		defer f.Close()

		http.ServeContent(w, r, o.Name, o.CreatedAt, f.(io.ReadSeeker))
		return
	}
	webutil.JSON(w, o, http.StatusOK)
}

func (h API) Destroy(u *auth.User, o *object.Object, w http.ResponseWriter, r *http.Request) {
	if err := h.Handler.Destroy(r.Context(), o); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to delete object"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func RegisterAPI(a auth.Authenticator, srv *server.Server) {
	api := API{
		Handler: NewHandler(srv),
	}

	index := api.Restrict(a, []string{"object:read"}, api.Index)
	store := api.Restrict(a, []string{"object:write"}, api.Store)

	a = namespace.NewAuth[*object.Object](a, "object", api.Objects.Store)

	show := api.Restrict(a, []string{"object:read"}, api.Object(api.Show))
	destroy := api.Restrict(a, []string{"object:delete"}, api.Object(api.Destroy))

	sr := srv.Router.PathPrefix("/objects").Subrouter()
	sr.HandleFunc("", index).Methods("GET")
	sr.HandleFunc("", store).Methods("POST")
	sr.HandleFunc("/{object:[0-9]+}", show).Methods("GET")
	sr.HandleFunc("/{object:[0-9]+}/builds", show).Methods("GET")
	sr.HandleFunc("/{object:[0-9]+}", destroy).Methods("DELETE")
}
