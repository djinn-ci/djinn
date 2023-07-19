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
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"

	"github.com/gorilla/mux"
)

type HandlerFunc func(*auth.User, *build.Build, http.ResponseWriter, *http.Request)

type Handler struct {
	*server.Server

	Builds     *build.Store
	Artifacts  *build.ArtifactStore
	Jobs       build.JobStore
	Objects    *object.Store
	Namespaces *database.Store[*namespace.Namespace]
	Variables  *database.Store[*build.Variable]
	Keys       *database.Store[*build.Key]
	Triggers   *database.Store[*build.Trigger]
	Tags       *database.Store[*build.Tag]
	Stages     *database.Store[*build.Stage]
	Users      *database.Store[*auth.User]
}

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server: srv,
		Builds: &build.Store{
			Store:  build.NewStore(srv.DB),
			Hasher: srv.Hasher,
			Queues: srv.DriverQueues,
		},
		Jobs: build.JobStore{
			Store: build.NewJobStore(srv.DB),
		},
		Objects: &object.Store{
			Store: object.NewStore(srv.DB),
		},
		Artifacts: &build.ArtifactStore{
			Store: build.NewArtifactStore(srv.DB),
			FS:    srv.Artifacts,
		},
		Namespaces: namespace.NewStore(srv.DB),
		Variables:  build.NewVariableStore(srv.DB),
		Keys:       build.NewKeyStore(srv.DB),
		Triggers:   build.NewTriggerStore(srv.DB),
		Tags:       build.NewTagStore(srv.DB),
		Stages:     build.NewStageStore(srv.DB),
		Users:      user.NewStore(srv.DB),
	}
}

func (h *Handler) Build(fn HandlerFunc) auth.HandlerFunc {
	return func(u *auth.User, w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		vars := mux.Vars(r)

		owner, ok, err := h.Users.Get(ctx, user.WhereUsername(vars["username"]))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get build owner"))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}

		b, ok, err := h.Builds.Get(
			ctx,
			query.Where("user_id", "=", query.Arg(owner.ID)),
			query.Where("number", "=", query.Arg(vars["build"])),
		)

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get build"))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}

		b.User = owner

		if !b.NamespaceID.Valid {
			if u.ID != owner.ID {
				h.NotFound(w, r)
				return
			}
			fn(u, b, w, r)
			return
		}

		root, ok, err := h.Namespaces.Get(
			ctx,
			query.Where("id", "=", namespace.Select(
				query.Columns("root_id"),
				query.Where("id", "=", query.Arg(b.NamespaceID)),
			)),
		)

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get root namespace"))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}

		b.Namespace = root

		if root.ID != b.NamespaceID.Elem {
			b.Namespace, _, err = h.Namespaces.Get(ctx, query.Where("id", "=", query.Arg(b.NamespaceID)))

			if err != nil {
				h.InternalServerError(w, r, errors.Wrap(err, "Failed to get namespace"))
				return
			}
		}

		b.Namespace.User = b.User

		if b.Namespace.User.ID != b.Namespace.UserID {
			u, _, err := h.Users.Get(ctx, user.WhereID(b.Namespace.UserID))

			if err != nil {
				h.InternalServerError(w, r, errors.Wrap(err, "Failed to get namespace owner"))
				return
			}
			b.Namespace.User = u
		}

		if err := root.HasAccess(ctx, h.DB, u); err != nil {
			if !errors.Is(err, auth.ErrPermission) {
				h.InternalServerError(w, r, errors.Wrap(err, "Failed to get root namespace"))
				return
			}

			h.NotFound(w, r)
			return
		}

		if root.Visibility != namespace.Public {
			if !u.Has("build:read") {
				h.NotFound(w, r)
				return
			}
		}
		fn(u, b, w, r)
	}
}

func (h *Handler) Index(u *auth.User, r *http.Request) (*database.Paginator[*build.Build], error) {
	ctx := r.Context()

	p, err := h.Builds.Index(ctx, r.URL.Query(), namespace.WhereCollaborator(u.ID))

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := build.LoadRelations(ctx, h.DB, p.Items...); err != nil {
		return nil, errors.Err(err)
	}
	return p, nil
}

func (h *Handler) Store(u *auth.User, r *http.Request) (*build.Build, *Form, error) {
	f := Form{
		DB:      h.DB,
		User:    u,
		Drivers: make(map[string]struct{}),
	}

	for driver := range h.DriverQueues {
		f.Drivers[driver] = struct{}{}
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		return nil, &f, errors.Err(err)
	}

	ctx := r.Context()

	p := build.Params{
		User: u,
		Trigger: &build.Trigger{
			Type:    build.Manual,
			Comment: f.Comment,
			Data: map[string]string{
				"email":    u.Email,
				"username": u.Username,
			},
		},
		Manifest: f.Manifest,
		Tags:     f.Tags,
	}

	b, err := h.Builds.Create(ctx, &p)

	if err != nil {
		pathError := &namespace.PathError{}

		if errors.As(err, &pathError) {
			return nil, &f, pathError
		}
		return nil, &f, errors.Err(err)
	}

	if err := h.Builds.Submit(ctx, h.Host, b); err != nil {
		return nil, &f, errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &build.Event{Build: b})
	return b, &f, nil
}

func (h *Handler) StoreTag(u *auth.User, b *build.Build, r *http.Request) ([]*build.Tag, error) {
	var f TagForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		return nil, errors.Err(err)
	}

	ctx := r.Context()

	if err := b.Tag(ctx, h.DB, u, f.Tags...); err != nil {
		return nil, errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &build.TagEvent{
		Build: b,
		User:  u,
		Tags:  b.Tags,
	})
	return b.Tags, nil
}

func (h *Handler) DestroyTag(ctx context.Context, b *build.Build, name string) error {
	t, ok, err := h.Tags.Get(
		ctx,
		query.Where("build_id", "=", query.Arg(b.ID)),
		query.Where("name", "=", query.Arg(name)),
	)

	if err != nil {
		return errors.Err(err)
	}

	if !ok {
		return database.ErrNoRows
	}

	if err := h.Tags.Delete(ctx, t); err != nil {
		return errors.Err(err)
	}
	return nil
}
