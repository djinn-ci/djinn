package http

import (
	"context"
	"net/http"

	"djinn-ci.com/build"
	"djinn-ci.com/cron"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
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
	Crons      cron.Store
	Builds     *build.Store
	Namespaces namespace.Store
	Users      user.Store
}

type HandlerFunc func(*user.User, *cron.Cron, http.ResponseWriter, *http.Request)

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server:     srv,
		Namespace:  namespacehttp.NewHandler(srv),
		Crons:      cron.Store{Pool: srv.DB},
		Builds:     &build.Store{Pool: srv.DB},
		Namespaces: namespace.Store{Pool: srv.DB},
		Users:      user.Store{Pool: srv.DB},
	}
}

func (h *Handler) WithCron(fn HandlerFunc) userhttp.HandlerFunc {
	return func(u *user.User, w http.ResponseWriter, r *http.Request) {
		var (
			c   *cron.Cron
			ok  bool
			err error
		)

		get := func(id int64) (database.Model, bool, error) {
			c, ok, err = h.Crons.Get(query.Where("id", "=", query.Arg(id)))

			if err != nil {
				return nil, false, errors.Err(err)
			}
			return c, ok, nil
		}

		ok, err = h.Namespace.CanAccessResource("cron", get, u, r)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}
		fn(u, c, w, r)
	}
}

func (h *Handler) IndexWithRelations(u *user.User, r *http.Request) ([]*cron.Cron, database.Paginator, error) {
	cc, paginator, err := h.Crons.Index(r.URL.Query(), namespace.WhereCollaborator(u.ID))

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	if err := cron.LoadRelations(h.DB, cc...); err != nil {
		return nil, paginator, errors.Err(err)
	}
	return cc, paginator, nil
}

func (h *Handler) StoreModel(u *user.User, r *http.Request) (*cron.Cron, Form, error) {
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
		Crons:  h.Crons,
		Form:   f,
	}

	if err := webutil.Validate(&v); err != nil {
		err.(webutil.ValidationErrors).Merge(verrs)

		return nil, f, errors.Err(err)
	}

	c, err := h.Crons.Create(cron.Params{
		UserID:   u.ID,
		Name:     f.Name,
		Schedule: f.Schedule,
		Manifest: f.Manifest,
	})

	if err != nil {
		cause := errors.Cause(err)

		if perr, ok := cause.(*namespace.PathError); ok {
			return nil, f, perr.Err
		}
		return nil, f, errors.Err(err)
	}

	if err := h.Users.Load("user_id", "id", c); err != nil {
		return nil, f, errors.Err(err)
	}

	c.Author = u

	h.Queues.Produce(r.Context(), "events", &cron.Event{
		Cron:   c,
		Action: "created",
	})

	return c, f, nil
}

func (h *Handler) UpdateModel(c *cron.Cron, r *http.Request) (*cron.Cron, Form, error) {
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
		UserID: c.UserID,
		Crons:  h.Crons,
		Cron:   c,
		Form:   f,
	}

	if err := webutil.Validate(&v); err != nil {
		err.(webutil.ValidationErrors).Merge(verrs)

		return nil, f, errors.Err(err)
	}

	if f.Name == "" {
		f.Name = c.Name
	}

	if f.Manifest.String() == "{}" {
		f.Manifest = c.Manifest
	}

	p := cron.Params{
		UserID:   c.UserID,
		Name:     f.Name,
		Schedule: f.Schedule,
		Manifest: f.Manifest,
	}

	if err := h.Crons.Update(c.ID, p); err != nil {
		cause := errors.Cause(err)

		if perr, ok := cause.(*namespace.PathError); ok {
			return nil, f, perr.Err
		}
		return nil, f, errors.Err(err)
	}

	c.Name = f.Name
	c.Schedule = f.Schedule
	c.Manifest = f.Manifest

	if err := h.Users.Load("user_id", "id", c); err != nil {
		return nil, f, errors.Err(err)
	}

	if err := h.Users.Load("author_id", "id", c); err != nil {
		return nil, f, errors.Err(err)
	}

	h.Queues.Produce(r.Context(), "events", &cron.Event{
		Cron:   c,
		Action: "updated",
	})
	return c, f, nil
}

func (h *Handler) DeleteModel(ctx context.Context, c *cron.Cron) error {
	if err := h.Users.Load("user_id", "id", c); err != nil {
		return errors.Err(err)
	}

	if err := h.Users.Load("author_id", "id", c); err != nil {
		return errors.Err(err)
	}

	if err := h.Crons.Delete(c.ID); err != nil {
		return errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &cron.Event{
		Cron:   c,
		Action: "deleted",
	})
	return nil
}
