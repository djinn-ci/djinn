package http

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"djinn-ci.com/database"
	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/image"
	"djinn-ci.com/namespace"
	namespacehttp "djinn-ci.com/namespace/http"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/mux"
)

type Handler struct {
	*server.Server

	Namespace  *namespacehttp.Handler
	Users      user.Store
	Images     *image.Store
	Downloads  image.DownloadStore
	Namespaces namespace.Store
}

type HandlerFunc func(*user.User, *image.Image, http.ResponseWriter, *http.Request)

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server:     srv,
		Namespace:  namespacehttp.NewHandler(srv),
		Namespaces: namespace.Store{Pool: srv.DB},
		Users:      user.Store{Pool: srv.DB},
		Images: &image.Store{
			Pool:   srv.DB,
			Store:  srv.Images,
			Hasher: srv.Hasher,
		},
		Downloads: image.DownloadStore{Pool: srv.DB},
	}
}

func (h *Handler) WithImage(fn HandlerFunc) userhttp.HandlerFunc {
	namespaces := namespace.Store{
		Pool: h.DB,
	}

	return func(u *user.User, w http.ResponseWriter, r *http.Request) {
		var hasPermission bool

		switch r.Method {
		case "GET":
			_, hasPermission = u.Permissions["image:read"]
		case "POST", "PATCH":
			_, hasPermission = u.Permissions["image:write"]
		case "DELETE":
			_, hasPermission = u.Permissions["image:delete"]
		}

		vars := mux.Vars(r)

		id, _ := strconv.ParseInt(vars["image"], 10, 64)

		i, ok, err := h.Images.Get(query.Where("id", "=", query.Arg(id)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}

		if !i.NamespaceID.Valid {
			if u.ID == i.UserID && hasPermission {
				fn(u, i, w, r)
				return
			}

			h.NotFound(w, r)
			return
		}

		root, _, err := namespaces.Get(
			query.Where("root_id", "=", namespace.SelectRootID(i.NamespaceID.Int64)),
			query.Where("id", "=", namespace.SelectRootID(i.NamespaceID.Int64)),
		)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if r.Method == "GET" {
			parts := strings.Split(r.URL.Path, "/")

			// Allow downloading of images from public namespaces.
			if parts[len(parts)-2] == "download" {
				if root.Visibility == namespace.Public {
					fn(u, i, w, r)
					return
				}
			}

			if err := root.HasAccess(h.DB, u.ID); err != nil {
				if errors.Is(err, namespace.ErrPermission) {
					h.NotFound(w, r)
					return
				}

				h.InternalServerError(w, r, errors.Err(err))
				return
			}
		}

		if err := root.IsCollaborator(h.DB, u.ID); err != nil {
			if errors.Is(err, namespace.ErrPermission) {
				h.NotFound(w, r)
				return
			}

			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if !hasPermission {
			h.NotFound(w, r)
			return
		}
		fn(u, i, w, r)
	}
}

func (h *Handler) IndexWithRelations(u *user.User, r *http.Request) ([]*image.Image, database.Paginator, error) {
	ii, paginator, err := h.Images.Index(r.URL.Query(), namespace.WhereCollaborator(u.ID))

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	if err := image.LoadRelations(h.DB, ii...); err != nil {
		return nil, paginator, errors.Err(err)
	}
	return ii, paginator, nil
}

func (h *Handler) StoreModel(u *user.User, r *http.Request) (*image.Image, Form, error) {
	f := Form{
		File: &webutil.File{
			Field: "file",
		},
	}

	var verrs webutil.ValidationErrors

	// Assume no image is being sent in the request body, so unmarshal the JSON
	// payload.
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		if err := webutil.UnmarshalForm(&f, r); err != nil {
			if verrs0, ok := err.(webutil.ValidationErrors); ok {
				verrs = verrs0
				goto validate
			}
			return nil, f, errors.Err(err)
		}
		goto validate
	}

	if err := webutil.UnmarshalFormWithFile(&f, f.File, r); err != nil {
		if verrs0, ok := err.(webutil.ValidationErrors); ok {
			verrs = verrs0
			goto validate
		}
		return nil, f, errors.Err(err)
	}

	defer f.File.Remove()

validate:
	v := Validator{
		UserID: u.ID,
		Images: h.Images,
		File: &webutil.FileValidator{
			File: f.File,
		},
		Form: &f,
	}

	if err := webutil.Validate(&v); err != nil {
		err.(webutil.ValidationErrors).Merge(verrs)

		return nil, f, errors.Err(err)
	}

	i, err := h.Images.Create(image.Params{
		UserID:    u.ID,
		Namespace: f.Namespace,
		Name:      f.Name,
		Driver:    driver.QEMU,
		Image:     f.File.File,
	})

	if err != nil {
		cause := errors.Cause(err)

		if perr, ok := cause.(*namespace.PathError); ok {
			return nil, f, perr.Err
		}
		return nil, f, errors.Err(err)
	}

	if f.DownloadURL.URL != nil {
		j, err := image.NewDownloadJob(h.DB, i, f.DownloadURL)

		if err != nil {
			return nil, f, errors.Err(err)
		}

		if _, err := h.Queues.Produce(r.Context(), "jobs", j); err != nil {
			return nil, f, errors.Err(err)
		}
	}

	if err := h.Users.Load("user_id", "id", i); err != nil {
		return nil, f, errors.Err(err)
	}

	i.Author = u

	h.Queues.Produce(r.Context(), "events", &image.Event{
		Image:  i,
		Action: "created",
	})
	return i, f, nil
}

func (h *Handler) DeleteModel(ctx context.Context, i *image.Image) error {
	if err := h.Images.Delete(i); err != nil {
		return errors.Err(err)
	}

	if err := h.Users.Load("user_id", "id", i); err != nil {
		return errors.Err(err)
	}

	if err := h.Users.Load("author_id", "id", i); err != nil {
		return errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &image.Event{
		Image:  i,
		Action: "deleted",
	})
	return nil
}
