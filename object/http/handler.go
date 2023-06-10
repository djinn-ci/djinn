package http

import (
	"context"
	"net/http"

	"djinn-ci.com/auth"
	"djinn-ci.com/build"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/object"
	"djinn-ci.com/server"

	"github.com/andrewpillar/fs"
	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"

	"github.com/gorilla/mux"
)

type Handler struct {
	*server.Server

	Objects *object.Store
	Builds  *build.Store
}

type HandlerFunc func(*auth.User, *object.Object, http.ResponseWriter, *http.Request)

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server: srv,
		Objects: &object.Store{
			Store:  object.NewStore(srv.DB),
			FS:     srv.Objects,
			Hasher: srv.Hasher,
		},
		Builds: &build.Store{Store: build.NewStore(srv.DB)},
	}
}

func (h *Handler) Object(fn HandlerFunc) auth.HandlerFunc {
	return func(u *auth.User, w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := mux.Vars(r)["object"]

		o, ok, err := h.Objects.Get(ctx, query.Where("id", "=", query.Arg(id)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get object"))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}

		if err := namespace.LoadResourceRelations[*object.Object](ctx, h.DB, o); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to load object relations"))
			return
		}
		fn(u, o, w, r)
	}
}

func (h *Handler) Index(u *auth.User, r *http.Request) (*database.Paginator[*object.Object], error) {
	ctx := r.Context()

	p, err := h.Objects.Index(ctx, r.URL.Query(), namespace.WhereCollaborator(u.ID))

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := namespace.LoadResourceRelations(ctx, h.DB, p.Items...); err != nil {
		return nil, errors.Err(err)
	}
	return p, nil
}

func (h *Handler) Store(u *auth.User, r *http.Request) (*object.Object, *Form, error) {
	ctx := r.Context()

	f := Form{
		Pool: h.DB,
		User: u,
	}

	file, _, err := webutil.UnmarshalFormWithFile(&f, "file", r)

	if err != nil {
		return nil, &f, errors.Err(err)
	}

	if file != nil {
		f.File = file.File
	}

	if err := f.Validate(ctx); err != nil {
		return nil, &f, errors.Err(err)
	}

	o, err := h.Objects.Create(ctx, &object.Params{
		User:      u,
		Namespace: f.Namespace,
		Name:      f.Name,
		Object:    f.File,
	})

	if err != nil {
		var sizeErr fs.SizeError

		if errors.As(err, &sizeErr) {
			return nil, &f, webutil.ValidationErrors{
				"file": []string{sizeErr.Error()},
			}
		}
		return nil, &f, errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &object.Event{
		Object: o,
		Action: "created",
	})
	return o, &f, nil
}

func (h *Handler) Destroy(ctx context.Context, o *object.Object) error {
	if err := h.Objects.Delete(ctx, o); err != nil {
		return errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &object.Event{
		Object: o,
		Action: "deleted",
	})
	return nil
}
