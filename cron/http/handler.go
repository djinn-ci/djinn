package http

import (
	"context"
	"net/http"

	"djinn-ci.com/auth"
	"djinn-ci.com/build"
	"djinn-ci.com/cron"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"

	"github.com/gorilla/mux"
)

type Handler struct {
	*server.Server

	Crons  cron.Store
	Builds *build.Store
}

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server: srv,
		Crons:  cron.Store{Store: cron.NewStore(srv.DB)},
		Builds: &build.Store{Store: build.NewStore(srv.DB)},
	}
}

type HandlerFunc func(*auth.User, *cron.Cron, http.ResponseWriter, *http.Request)

func (h *Handler) Cron(fn HandlerFunc) auth.HandlerFunc {
	return func(u *auth.User, w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := mux.Vars(r)["cron"]

		c, ok, err := h.Crons.Get(ctx, query.Where("id", "=", query.Arg(id)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get cron job"))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}

		if err := namespace.LoadResourceRelations[*cron.Cron](ctx, h.DB, c); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to load cron job relations"))
			return
		}
		fn(u, c, w, r)
	}
}

func (h *Handler) Index(u *auth.User, r *http.Request) (*database.Paginator[*cron.Cron], error) {
	ctx := r.Context()

	p, err := h.Crons.Index(ctx, r.URL.Query(), namespace.WhereCollaborator(u.ID))

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := namespace.LoadResourceRelations(ctx, h.DB, p.Items...); err != nil {
		return nil, errors.Err(err)
	}
	return p, nil
}

func (h *Handler) Store(u *auth.User, r *http.Request) (*cron.Cron, *Form, error) {
	f := Form{
		Pool: h.DB,
		User: u,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		return nil, &f, errors.Err(err)
	}

	ctx := r.Context()

	c, err := h.Crons.Create(ctx, &cron.Params{
		User:     u,
		Name:     f.Name,
		Schedule: f.Schedule,
		Manifest: f.Manifest,
	})

	if err != nil {
		return nil, &f, errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &cron.Event{
		Cron:   c,
		Action: "created",
	})

	return c, &f, nil
}

func (h *Handler) Update(u *auth.User, c *cron.Cron, r *http.Request) (*cron.Cron, *Form, error) {
	f := Form{
		Pool: h.DB,
		User: u,
		Cron: c,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		return nil, &f, errors.Err(err)
	}

	ctx := r.Context()

	c.Name = f.Name
	c.Schedule = f.Schedule
	c.Manifest = f.Manifest

	if err := h.Crons.Update(ctx, c); err != nil {
		return nil, &f, errors.Err(err)
	}

	ld := user.Loader(h.DB)

	if err := ld.Load(ctx, "user_id", "id", c); err != nil {
		return nil, &f, errors.Err(err)
	}
	if err := ld.Load(ctx, "author_id", "id", c); err != nil {
		return nil, &f, errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &cron.Event{
		Cron:   c,
		Action: "updated",
	})
	return c, &f, nil
}

func (h *Handler) Destroy(ctx context.Context, c *cron.Cron) error {
	if err := h.Crons.Delete(ctx, c); err != nil {
		return errors.Err(err)
	}

	ld := user.Loader(h.DB)

	if err := ld.Load(ctx, "user_id", "id", c); err != nil {
		return errors.Err(err)
	}
	if err := ld.Load(ctx, "author_id", "id", c); err != nil {
		return errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &cron.Event{
		Cron:   c,
		Action: "deleted",
	})
	return nil
}
