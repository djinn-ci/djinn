package http

import (
	"context"
	"net/http"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/key"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"

	"github.com/gorilla/mux"
)

type Handler struct {
	*server.Server

	Keys *key.Store
}

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server: srv,
		Keys: &key.Store{
			Store:  key.NewStore(srv.DB),
			AESGCM: srv.AESGCM,
		},
	}
}

type HandlerFunc func(*auth.User, *key.Key, http.ResponseWriter, *http.Request)

func (h *Handler) Key(fn HandlerFunc) auth.HandlerFunc {
	return func(u *auth.User, w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id := mux.Vars(r)["key"]

		k, ok, err := h.Keys.Get(ctx, query.Where("id", "=", query.Arg(id)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get key"))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}

		if err := namespace.LoadResourceRelations[*key.Key](ctx, h.DB, k); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to load key relations"))
			return
		}
		fn(u, k, w, r)
	}
}

func (h *Handler) Index(u *auth.User, r *http.Request) (*database.Paginator[*key.Key], error) {
	ctx := r.Context()

	p, err := h.Keys.Index(ctx, r.URL.Query(), namespace.WhereCollaborator(u.ID))

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := namespace.LoadResourceRelations(ctx, h.DB, p.Items...); err != nil {
		return nil, errors.Err(err)
	}
	return p, nil
}

func (h *Handler) Store(u *auth.User, r *http.Request) (*key.Key, *Form, error) {
	f := Form{
		Pool: h.DB,
		User: u,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		return nil, &f, err
	}

	ctx := r.Context()

	k, err := h.Keys.Create(ctx, &key.Params{
		User:      u,
		Namespace: f.Namespace,
		Name:      f.Name,
		Key:       f.SSHKey,
		Config:    f.Config,
	})

	if err != nil {
		return nil, &f, errors.Err(err)
	}

	h.Queues.Produce(r.Context(), "events", &key.Event{
		Key:    k,
		Action: "created",
	})
	return k, &f, nil
}

func (h *Handler) Update(u *auth.User, k *key.Key, r *http.Request) (*key.Key, *Form, error) {
	f := Form{
		Pool: h.DB,
		User: u,
		Key:  k,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		return nil, &f, err
	}

	ctx := r.Context()

	k.Config = f.Config

	if err := h.Keys.Update(ctx, k); err != nil {
		return nil, &f, errors.Err(err)
	}

	h.Queues.Produce(r.Context(), "events", &key.Event{
		Key:    k,
		Action: "updated",
	})
	return k, &f, nil
}

func (h *Handler) Destroy(ctx context.Context, k *key.Key) error {
	if err := h.Keys.Delete(ctx, k); err != nil {
		return errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &key.Event{
		Key:    k,
		Action: "deleted",
	})
	return nil
}
