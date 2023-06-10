package http

import (
	"context"
	"net/http"
	"strings"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/image"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"

	"github.com/gorilla/mux"
)

type Handler struct {
	*server.Server

	Images    *image.Store
	Downloads *database.Store[*image.Download]
}

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server: srv,
		Images: &image.Store{
			Store:  image.NewStore(srv.DB),
			FS:     srv.Images,
			Hasher: srv.Hasher,
		},
		Downloads: image.NewDownloadStore(srv.DB),
	}
}

type HandlerFunc func(*auth.User, *image.Image, http.ResponseWriter, *http.Request)

func (h *Handler) Image(fn HandlerFunc) auth.HandlerFunc {
	return func(u *auth.User, w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["image"]

		i, ok, err := h.Images.Get(r.Context(), query.Where("id", "=", query.Arg(id)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get image"))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}

		if !i.NamespaceID.Valid {
			if u.ID != i.UserID {
				h.NotFound(w, r)
				return
			}
		}

		if !u.Has("image:read") {
			h.NotFound(w, r)
			return
		}
		fn(u, i, w, r)
	}
}

func (h *Handler) Index(u *auth.User, r *http.Request) (*database.Paginator[*image.Image], error) {
	ctx := r.Context()

	p, err := h.Images.Index(ctx, r.URL.Query(), namespace.WhereCollaborator(u.ID))

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := image.LoadRelations(ctx, h.DB, p.Items...); err != nil {
		return nil, errors.Err(err)
	}
	return p, nil
}

func (h *Handler) Store(u *auth.User, r *http.Request) (*image.Image, *Form, error) {
	ctx := r.Context()

	f := Form{
		Pool: h.DB,
		User: u,
	}

	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		if err := webutil.UnmarshalForm(&f, r); err != nil {
			return nil, &f, errors.Err(err)
		}
	} else {
		file, _, err := webutil.UnmarshalFormWithFile(&f, "file", r)

		if err != nil {
			return nil, &f, errors.Err(err)
		}

		if file != nil {
			f.File = file.File
		}
	}

	if err := f.Validate(ctx); err != nil {
		return nil, &f, errors.Err(err)
	}

	i, err := h.Images.Create(ctx, &image.Params{
		User:      u,
		Namespace: f.Namespace,
		Name:      f.Name,
		Driver:    driver.QEMU,
		Image:     f.File,
	})

	if err != nil {
		return nil, &f, errors.Err(err)
	}

	if f.DownloadURL.URL != nil {
		j, err := image.NewDownloadJob(h.DB, i, f.DownloadURL)

		if err != nil {
			return nil, &f, errors.Err(err)
		}

		if _, err := h.Queues.Produce(ctx, "jobs", j); err != nil {
			return nil, &f, errors.Err(err)
		}
	}

	h.Queues.Produce(r.Context(), "events", &image.Event{
		Image:  i,
		Action: "created",
	})
	return i, &f, nil
}

func (h *Handler) Destroy(ctx context.Context, i *image.Image) error {
	if err := h.Images.Delete(ctx, i); err != nil {
		return errors.Err(err)
	}

	if err := image.LoadRelations(ctx, h.DB, i); err != nil {
		return errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &image.Event{
		Image:  i,
		Action: "deleted",
	})
	return nil
}
