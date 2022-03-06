package http

import (
	"context"
	"net/http"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	namespacehttp "djinn-ci.com/namespace/http"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

type Handler struct {
	*server.Server

	Namespace  *namespacehttp.Handler
	Namespaces namespace.Store
	Variables  variable.Store
	Users      user.Store
}

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server:     srv,
		Namespace:  namespacehttp.NewHandler(srv),
		Namespaces: namespace.Store{Pool: srv.DB},
		Variables: variable.Store{
			Pool:   srv.DB,
			AESGCM: srv.AESGCM,
		},
		Users: user.Store{Pool: srv.DB},
	}
}

type HandlerFunc func(*user.User, *variable.Variable, http.ResponseWriter, *http.Request)

func (h *Handler) WithVariable(fn HandlerFunc) userhttp.HandlerFunc {
	return func(u *user.User, w http.ResponseWriter, r *http.Request) {
		var (
			v   *variable.Variable
			ok  bool
			err error
		)

		get := func(id int64) (database.Model, bool, error) {
			v, ok, err = h.Variables.Get(query.Where("id", "=", query.Arg(id)))

			if err != nil {
				return nil, false, errors.Err(err)
			}
			return v, ok, nil
		}

		ok, err = h.Namespace.CanAccessResource("variable", get, u, r)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}
		fn(u, v, w, r)
	}
}

func (h *Handler) IndexWithRelations(u *user.User, r *http.Request) ([]*variable.Variable, database.Paginator, error) {
	vv, paginator, err := h.Variables.Index(r.URL.Query(), namespace.WhereCollaborator(u.ID))

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	if err := variable.LoadRelations(h.DB, vv...); err != nil {
		return nil, paginator, errors.Err(err)
	}
	return vv, paginator, nil
}

func (h *Handler) StoreModel(u *user.User, r *http.Request) (*variable.Variable, Form, error) {
	var f Form

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	val := Validator{
		UserID:    u.ID,
		Variables: h.Variables,
		Form:      f,
	}

	if err := webutil.Validate(&val); err != nil {
		return nil, f, errors.Err(err)
	}

	v, err := h.Variables.Create(variable.Params{
		UserID:    u.ID,
		Namespace: f.Namespace,
		Key:       f.Key,
		Value:     f.Value,
		Masked:    f.Mask,
	})

	if err != nil {
		cause := errors.Cause(err)

		if perr, ok := cause.(*namespace.PathError); ok {
			return nil, f, perr.Err
		}
		return nil, f, errors.Err(err)
	}

	if err := h.Users.Load("user_id", "id", v); err != nil {
		return nil, f, errors.Err(err)
	}

	v.Author = u

	h.Queues.Produce(r.Context(), "events", &variable.Event{
		Variable: v,
		Action:   "created",
	})
	return v, f, nil
}

func (h *Handler) DeleteModel(ctx context.Context, v *variable.Variable) error {
	if err := h.Variables.Delete(v.ID); err != nil {
		return errors.Err(err)
	}

	if err := h.Users.Load("user_id", "id", v); err != nil {
		return errors.Err(err)
	}

	if err := h.Users.Load("author_id", "id", v); err != nil {
		return errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &variable.Event{
		Variable: v,
		Action:   "deleted",
	})
	return nil
}
