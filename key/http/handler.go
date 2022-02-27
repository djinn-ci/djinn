package http

import (
	"context"
	"net/http"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/key"
	"djinn-ci.com/namespace"
	namespacehttp "djinn-ci.com/namespace/http"
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
	Keys       *key.Store
}

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server:    srv,
		Namespace: namespacehttp.NewHandler(srv),
		Users:     user.Store{Pool: srv.DB},
		Keys: &key.Store{
			Pool:   srv.DB,
			AESGCM: srv.AESGCM,
		},
	}
}

type HandlerFunc func(*user.User, *key.Key, http.ResponseWriter, *http.Request)

func (h *Handler) WithKey(fn HandlerFunc) userhttp.HandlerFunc {
	return func(u *user.User, w http.ResponseWriter, r *http.Request) {
		var (
			k   *key.Key
			ok  bool
			err error
		)

		get := func(id int64) (database.Model, bool, error) {
			k, ok, err = h.Keys.Get(query.Where("id", "=", query.Arg(id)))

			if err != nil {
				return nil, false, errors.Err(err)
			}
			return k, ok, nil
		}

		ok, err = h.Namespace.CanAccessResource("key", get, u, r)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}
		fn(u, k, w, r)
	}
}

func (h *Handler) IndexWithRelations(u *user.User, r *http.Request) ([]*key.Key, database.Paginator, error) {
	kk, paginator, err := h.Keys.Index(r.URL.Query(), namespace.WhereCollaborator(u.ID))

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	mm := make([]database.Model, 0, len(kk))

	for _, k := range kk {
		mm = append(mm, k)
	}

	if err := h.Users.Load("user_id", "id", mm...); err != nil {
		return nil, paginator, errors.Err(err)
	}

	if err := h.Users.Load("author_id", "id", mm...); err != nil {
		return nil, paginator, errors.Err(err)
	}
	return kk, paginator, nil
}

func (h *Handler) StoreModel(u *user.User, r *http.Request) (*key.Key, Form, error) {
	var (
		f     Form
		verrs webutil.ValidationErrors
	)

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		if verrs0, ok := err.(webutil.ValidationErrors); ok {
			verrs = verrs0
			goto validate
		}
		return nil, f, errors.Err(err)
	}

validate:
	v := Validator{
		UserID: u.ID,
		Keys:   h.Keys,
		Form:   f,
	}

	if err := webutil.Validate(&v); err != nil {
		err.(webutil.ValidationErrors).Merge(verrs)

		return nil, f, errors.Err(err)
	}

	k, err := h.Keys.Create(key.Params{
		UserID:    u.ID,
		Namespace: f.Namespace,
		Name:      f.Name,
		Key:       f.Key,
		Config:    f.Config,
	})

	if err != nil {
		cause := errors.Cause(err)

		if perr, ok := cause.(*namespace.PathError); ok {
			return nil, f, perr.Err
		}
		return nil, f, errors.Err(err)
	}

	if err := h.Users.Load("user_id", "id", k); err != nil {
		return nil, f, errors.Err(err)
	}

	k.Author = u

	h.Queues.Produce(r.Context(), "events", &key.Event{
		Key:    k,
		Action: "created",
	})

	return k, f, nil
}

func (h *Handler) UpdateModel(k *key.Key, r *http.Request) (*key.Key, Form, error) {
	var (
		f     Form
		verrs webutil.ValidationErrors
	)

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		if verrs0, ok := err.(webutil.ValidationErrors); ok {
			verrs = verrs0
			goto validate
		}
		return nil, f, errors.Err(err)
	}

validate:
	v := Validator{
		UserID: k.UserID,
		Keys:   h.Keys,
		Key:    k,
		Form:   f,
	}

	if err := webutil.Validate(&v); err != nil {
		err.(webutil.ValidationErrors).Merge(verrs)

		return nil, f, errors.Err(err)
	}

	if err := h.Keys.Update(k.ID, key.Params{Config: f.Config}); err != nil {
		return nil, f, errors.Err(err)
	}

	k.Config = f.Config

	if err := h.Users.Load("user_id", "id", k); err != nil {
		return nil, f, errors.Err(err)
	}

	if err := h.Users.Load("author_id", "id", k); err != nil {
		return nil, f, errors.Err(err)
	}

	h.Queues.Produce(r.Context(), "events", &key.Event{
		Key:    k,
		Action: "updated",
	})
	return k, f, nil
}

func (h *Handler) DeleteModel(ctx context.Context, k *key.Key) error {
	if err := h.Users.Load("user_id", "id", k); err != nil {
		return errors.Err(err)
	}

	if err := h.Users.Load("author_id", "id", k); err != nil {
		return errors.Err(err)
	}

	if err := h.Keys.Delete(k.ID); err != nil {
		return errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &key.Event{
		Key:    k,
		Action: "deleted",
	})
	return nil
}
