package http

import (
	"context"
	"net/http"

	"djinn-ci.com/build"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	namespacehttp "djinn-ci.com/namespace/http"
	"djinn-ci.com/object"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

type Handler struct {
	*server.Server

	Namespace  *namespacehttp.Handler
	Namespaces namespace.Store
	Users      user.Store
	Objects    *object.Store
	Builds     *build.Store
}

type HandlerFunc func(*user.User, *object.Object, http.ResponseWriter, *http.Request)

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server:     srv,
		Namespace:  namespacehttp.NewHandler(srv),
		Namespaces: namespace.Store{Pool: srv.DB},
		Users:      user.Store{Pool: srv.DB},
		Objects: &object.Store{
			Pool:   srv.DB,
			Store:  srv.Objects,
			Hasher: srv.Hasher,
		},
		Builds: &build.Store{Pool: srv.DB},
	}
}

func (h *Handler) WithObject(fn HandlerFunc) userhttp.HandlerFunc {
	return func(u *user.User, w http.ResponseWriter, r *http.Request) {
		var (
			o   *object.Object
			ok  bool
			err error
		)

		get := func(id int64) (database.Model, bool, error) {
			o, ok, err = h.Objects.Get(query.Where("id", "=", query.Arg(id)))

			if err != nil {
				return nil, false, errors.Err(err)
			}
			return o, ok, nil
		}

		ok, err = h.Namespace.CanAccessResource("object", get, u, r)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}
		fn(u, o, w, r)
	}
}

func (h *Handler) IndexWithRelations(u *user.User, r *http.Request) ([]*object.Object, database.Paginator, error) {
	oo, paginator, err := h.Objects.Index(r.URL.Query(), namespace.WhereCollaborator(u.ID))

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	if err := object.LoadRelations(h.DB, oo...); err != nil {
		return nil, paginator, errors.Err(err)
	}
	return oo, paginator, nil
}

func (h *Handler) StoreModel(u *user.User, r *http.Request) (*object.Object, Form, error) {
	f := Form{
		File: &webutil.File{
			Field: "file",
		},
	}

	var verrs webutil.ValidationErrors

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
		UserID:  u.ID,
		Objects: h.Objects,
		File: &webutil.FileValidator{
			File: f.File,
			Size: h.Objects.Limit(),
		},
		Form: &f,
	}

	if err := webutil.Validate(v); err != nil {
		err.(webutil.ValidationErrors).Merge(verrs)

		return nil, f, errors.Err(err)
	}

	o, err := h.Objects.Create(object.Params{
		UserID:    u.ID,
		Namespace: f.Namespace,
		Name:      f.Name,
		Object:    f.File.File,
	})

	if err != nil {
		cause := errors.Cause(err)

		if perr, ok := cause.(*namespace.PathError); ok {
			return nil, f, perr.Err
		}
		return nil, f, errors.Err(err)
	}

	if err := h.Users.Load("user_id", "id", o); err != nil {
		return nil, f, errors.Err(err)
	}

	o.Author = u

	h.Queues.Produce(r.Context(), "events", &object.Event{
		Object: o,
		Action: "created",
	})
	return o, f, nil
}

func (h *Handler) getBuilds(o *object.Object, r *http.Request) ([]*build.Build, database.Paginator, error) {
	bb, paginator, err := h.Builds.Index(
		r.URL.Query(), query.Where("id", "IN", build.SelectBuildIDForObject(o.ID)),
	)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	if err := build.LoadRelations(h.DB, bb...); err != nil {
		return nil, paginator, errors.Err(err)
	}
	return bb, paginator, nil
}

func (h *Handler) DeleteModel(ctx context.Context, o *object.Object) error {
	if err := h.Objects.Delete(o); err != nil {
		return errors.Err(err)
	}

	if err := h.Users.Load("user_id", "id", o); err != nil {
		return errors.Err(err)
	}

	if err := h.Users.Load("author_id", "id", o); err != nil {
		return errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &object.Event{
		Object: o,
		Action: "deleted",
	})
	return nil
}
