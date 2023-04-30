package http

import (
	"context"
	"net/http"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"

	"github.com/gorilla/mux"
)

type Handler struct {
	*server.Server

	Variables *variable.Store
}

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server: srv,
		Variables: &variable.Store{
			Store:  variable.NewStore(srv.DB),
			AESGCM: srv.AESGCM,
		},
	}
}

type HandlerFunc func(*auth.User, *variable.Variable, http.ResponseWriter, *http.Request)

func (h *Handler) Variable(fn HandlerFunc) auth.HandlerFunc {
	return func(u *auth.User, w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["variable"]

		v, ok, err := h.Variables.Get(r.Context(), query.Where("id", "=", query.Arg(id)))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get variable"))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}
		fn(u, v, w, r)
	}
}

func (h *Handler) Index(u *auth.User, r *http.Request) (*database.Paginator[*variable.Variable], error) {
	ctx := r.Context()

	p, err := h.Variables.Index(ctx, r.URL.Query(), namespace.WhereCollaborator(u.ID))

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := namespace.LoadResourceRelations(ctx, h.DB, p.Items...); err != nil {
		return nil, errors.Err(err)
	}
	return p, nil
}

func (h *Handler) Store(u *auth.User, r *http.Request) (*variable.Variable, Form, error) {
	f := Form{
		Pool: h.DB,
		User: u,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		return nil, f, err
	}

	ctx := r.Context()

	v, err := h.Variables.Create(ctx, variable.Params{
		User:      u,
		Namespace: f.Namespace,
		Key:       f.Key,
		Value:     f.Value,
		Masked:    f.Mask,
	})

	if err != nil {
		return nil, f, errors.Err(err)
	}

	h.Queues.Produce(r.Context(), "events", &variable.Event{
		Variable: v,
		Action:   "created",
	})
	return v, f, nil
}

func (h *Handler) Destroy(ctx context.Context, v *variable.Variable) error {
	if err := h.Variables.Delete(ctx, v); err != nil {
		return errors.Err(err)
	}

	ld := user.Loader(h.DB)

	if err := ld.Load(ctx, "user_id", "id", v); err != nil {
		return errors.Err(err)
	}
	if err := ld.Load(ctx, "author_id", "id", v); err != nil {
		return errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &variable.Event{
		Variable: v,
		Action:   "deleted",
	})
	return nil
}
