package http

import (
	"net/http"
	"strconv"

	"djinn-ci.com/build"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/object"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/mux"
)

type HandlerFunc func(*user.User, *build.Build, http.ResponseWriter, *http.Request)

type Handler struct {
	*server.Server

	Users      user.Store
	Builds     *build.Store
	Namespaces namespace.Store
	Jobs       build.JobStore
	Objects    *build.ObjectStore
	Artifacts  *build.ArtifactStore
	Variables  build.VariableStore
	Keys       build.KeyStore
	Tags       build.TagStore
	Triggers   build.TriggerStore
	Stages     build.StageStore
}

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server: srv,
		Users:  user.Store{Pool: srv.DB},
		Builds: &build.Store{
			Pool:         srv.DB,
			Hasher:       srv.Hasher,
			DriverQueues: srv.DriverQueues,
		},
		Namespaces: namespace.Store{Pool: srv.DB},
		Jobs:       build.JobStore{Pool: srv.DB},
		Objects: &build.ObjectStore{
			Pool:  srv.DB,
			Store: srv.Objects,
		},
		Artifacts: &build.ArtifactStore{
			Pool:  srv.DB,
			Store: srv.Artifacts,
		},
		Variables: build.VariableStore{Pool: srv.DB},
		Keys:      build.KeyStore{Pool: srv.DB},
		Tags:      build.TagStore{Pool: srv.DB},
		Triggers:  build.TriggerStore{Pool: srv.DB},
		Stages:    build.StageStore{Pool: srv.DB},
	}
}

func (h *Handler) WithBuild(fn HandlerFunc) userhttp.HandlerFunc {
	return func(u *user.User, w http.ResponseWriter, r *http.Request) {
		var ok bool

		switch r.Method {
		case "GET":
			_, ok = u.Permissions["build:read"]
		case "POST", "PATCH":
			_, ok = u.Permissions["build:write"]
		case "DELET":
			_, ok = u.Permissions["build:delete"]
		}

		vars := mux.Vars(r)

		owner, ok, err := h.Users.Get(user.WhereUsername(vars["username"]))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}

		number, _ := strconv.ParseInt(vars["build"], 10, 64)

		b, ok, err := h.Builds.Get(
			query.Where("user_id", "=", query.Arg(owner.ID)),
			query.Where("number", "=", query.Arg(number)),
		)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}

		// Not in a namespace, so check to see if the current user is the owner
		// of the build then defer to the handler.
		if !b.NamespaceID.Valid {
			if !(ok && u.ID == owner.ID) {
				h.NotFound(w, r)
				return
			}
			fn(u, b, w, r)
			return
		}

		n, ok, err := h.Namespaces.Get(query.Where("id", "=", query.Arg(b.NamespaceID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}

		root := n

		b.Namespace = n

		if n.RootID.Int64 != n.ID {
			root0, ok, err := h.Namespaces.Get(query.Where("id", "=", query.Arg(n.RootID)))

			if err != nil {
				h.InternalServerError(w, r, errors.Err(err))
				return
			}

			if !ok {
				h.NotFound(w, r)
				return
			}
			root = root0
		}

		if err := root.HasAccess(h.DB, u.ID); err != nil {
			if !errors.Is(err, namespace.ErrPermission) {
				h.InternalServerError(w, r, errors.Err(err))
				return
			}

			h.NotFound(w, r)
			return
		}
		fn(u, b, w, r)
	}
}

func (h *Handler) IndexWithRelations(u *user.User, r *http.Request) ([]*build.Build, database.Paginator, error) {
	bb, paginator, err := h.Builds.Index(r.URL.Query(), namespace.WhereCollaborator(u.ID))

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	if err := build.LoadRelations(h.DB, bb...); err != nil {
		return nil, paginator, errors.Err(err)
	}
	return bb, paginator, nil
}

func (h *Handler) StoreModel(u *user.User, r *http.Request) (*build.Build, Form, error) {
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
	drivers := make(map[string]struct{})

	for driver := range h.DriverQueues {
		drivers[driver] = struct{}{}
	}

	v := Validator{
		Form:    f,
		Drivers: drivers,
	}

	if err := webutil.Validate(&v); err != nil {
		err.(webutil.ValidationErrors).Merge(verrs)
		return nil, f, errors.Err(err)
	}

	b, err := h.Builds.Create(build.Params{
		UserID: u.ID,
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
	})

	if err != nil {
		cause := errors.Cause(err)

		if perr, ok := cause.(*namespace.PathError); ok {
			return nil, f, perr.Err
		}
		return nil, f, errors.Err(err)
	}

	ctx := r.Context()

	if err := h.Builds.Submit(ctx, webutil.BaseAddress(r), b); err != nil {
		return nil, f, errors.Err(err)
	}

	b.User = u

	if b.UserID != u.ID {
		if err := h.Users.Load("user_id", "id", b); err != nil {
			return nil, f, errors.Err(err)
		}
	}

	h.Queues.Produce(ctx, "events", &build.Event{Build: b})

	return b, f, nil
}

func (h *Handler) objectsWithRelations(b *build.Build) ([]*build.Object, error) {
	oo, err := h.Objects.All(query.Where("build_id", "=", query.Arg(b.ID)))

	if err != nil {
		return nil, errors.Err(err)
	}

	if len(oo) == 0 {
		return nil, nil
	}

	mm := make([]database.Model, 0, len(oo))

	for _, o := range oo {
		mm = append(mm, o)
	}

	objects := object.Store{Pool: h.DB}

	if err := objects.Load("object_id", "id", mm...); err != nil {
		return nil, errors.Err(err)
	}
	return oo, nil
}

func (h *Handler) variablesWithRelations(b *build.Build) ([]*build.Variable, error) {
	vv, err := h.Variables.All(query.Where("build_id", "=", query.Arg(b.ID)))

	if err != nil {
		return nil, errors.Err(err)
	}

	if len(vv) == 0 {
		return nil, nil
	}

	mm := make([]database.Model, 0, len(vv))

	for _, v := range vv {
		mm = append(mm, v)
	}

	variables := variable.Store{Pool: h.DB}

	if err := variables.Load("variable_id", "id", mm...); err != nil {
		return nil, errors.Err(err)
	}
	return vv, nil
}

func (h *Handler) StoreTagModel(u *user.User, b *build.Build, r *http.Request) ([]*build.Tag, error) {
	var f TagForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		return nil, errors.Err(err)
	}

	tt, err := h.Tags.Create(build.TagParams{
		BuildID: b.ID,
		UserID:  u.ID,
		Tags:    f.Tags,
	})

	if err != nil {
		return nil, errors.Err(err)
	}

	b.User = u

	if b.UserID != u.ID {
		if err := h.Users.Load("user_id", "id", b); err != nil {
			return nil, errors.Err(err)
		}
	}

	if b.NamespaceID.Valid {
		if err := h.Namespaces.Load("namespace_id", "id", b); err != nil {
			return nil, errors.Err(err)
		}
	}

	for _, t := range tt {
		t.Build = b
	}

	h.Queues.Produce(r.Context(), "events", &build.TagEvent{
		Build: b,
		User:  u,
		Tags:  tt,
	})

	return tt, nil
}

func (h *Handler) DeleteTagModel(b *build.Build, vars map[string]string) error {
	t, ok, err := h.Tags.Get(
		query.Where("build_id", "=", query.Arg(b.ID)),
		query.Where("name", "=", query.Arg(vars["name"])),
	)

	if err != nil {
		return errors.Err(err)
	}

	if !ok {
		return database.ErrNotFound
	}

	if err := h.Tags.Delete(t.ID); err != nil {
		return errors.Err(err)
	}
	return nil
}
