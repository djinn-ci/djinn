package http

import (
	"io"
	"net/http"

	"djinn-ci.com/auth"
	"djinn-ci.com/build"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/server"
	"djinn-ci.com/user"

	"github.com/andrewpillar/fs"
	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"

	"github.com/gorilla/mux"
)

type API struct {
	*Handler
}

func (h API) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	p, err := h.Handler.Index(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get builds"))
		return
	}

	w.Header().Set("Link", p.EncodeToLink(r.URL))
	webutil.JSON(w, p.Items, http.StatusOK)
}

func (h API) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	b, _, err := h.Handler.Store(u, r)

	if err != nil {
		h.FormError(w, r, nil, errors.Wrap(err, "Failed to submit build"))
		return
	}
	webutil.JSON(w, b, http.StatusCreated)
}

func (h API) Show(u *auth.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := build.LoadRelations(ctx, h.DB, b); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to load build relations"))
		return
	}

	switch webutil.BasePath(r.URL.Path) {
	case "objects":
		p, err := h.Objects.Index(ctx, r.URL.Query(), query.Where("id", "IN", build.SelectObject(
			query.Columns("object_id"),
			query.Where("build_id", "=", query.Arg(b.ID)),
		)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get objects"))
			return
		}

		w.Header().Set("Link", p.EncodeToLink(r.URL))
		webutil.JSON(w, p.Items, http.StatusOK)
		return
	case "artifacts":
		p, err := h.Artifacts.Index(ctx, r.URL.Query(), query.Where("build_id", "=", query.Arg(b.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get artifacts"))
			return
		}

		for _, a := range p.Items {
			a.Build = b
		}

		w.Header().Set("Link", p.EncodeToLink(r.URL))
		webutil.JSON(w, p.Items, http.StatusOK)
		return
	case "variables":
		vv, err := h.Variables.All(ctx, query.Where("build_id", "=", query.Arg(b.ID)), query.OrderAsc("key"))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get variables"))
			return
		}

		webutil.JSON(w, vv, http.StatusOK)
		return
	case "keys":
		kk, err := h.Keys.All(ctx, query.Where("build_id", "=", query.Arg(b.ID)), query.OrderAsc("name"))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get keys"))
			return
		}

		webutil.JSON(w, kk, http.StatusOK)
		return
	case "jobs":
		jj, err := h.Jobs.Index(ctx, r.URL.Query(), query.Where("build_id", "=", query.Arg(b.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get jobs"))
			return
		}

		if err := build.LoadJobRelations(ctx, h.DB, jj...); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to load job relations"))
			return
		}

		for _, j := range jj {
			j.Build = b
		}

		webutil.JSON(w, jj, http.StatusOK)
		return
	case "tags":
		tt, err := h.Tags.All(ctx, query.Where("build_id", "=", query.Arg(b.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get tags"))
			return
		}

		for _, t := range tt {
			t.Build = b
		}

		webutil.JSON(w, tt, http.StatusOK)
		return
	}
	webutil.JSON(w, b, http.StatusOK)
}

func (h API) ShowJob(u *auth.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	j, ok, err := h.Jobs.Get(
		ctx,
		query.Where("build_id", "=", query.Arg(b.ID)),
		query.Where("name", "=", query.Arg(mux.Vars(r)["name"])),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get job"))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	j.Build = b

	if err := build.LoadJobRelations(ctx, h.DB, j); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to load job relations"))
		return
	}
	webutil.JSON(w, j, http.StatusOK)
}

func (h API) Destroy(u *auth.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	if err := b.Kill(h.Redis); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to kill build"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h API) TogglePin(u *auth.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	switch webutil.BasePath(r.URL.Path) {
	case "pin":
		err = b.Pin(ctx, h.DB)
	case "unpin":
		err = b.Unpin(ctx, h.DB)
	}

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to update pin"))
		return
	}

	h.Queues.Produce(ctx, "events", &build.PinEvent{Build: b})
	webutil.JSON(w, b, http.StatusOK)
}

func (h API) Download(u *auth.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	a, ok, err := h.Artifacts.Get(
		r.Context(),
		query.Where("build_id", "=", query.Arg(b.ID)),
		query.Where("name", "=", query.Arg(name)),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	if r.Header.Get("Accept") == "application/octet-stream" {
		f, err := a.Open(h.Artifacts.FS)

		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				h.NotFound(w, r)
				return
			}

			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get artifact"))
			return
		}

		defer f.Close()
		http.ServeContent(w, r, a.Name, a.CreatedAt, f.(io.ReadSeeker))
		return
	}

	a.Build = b

	webutil.JSON(w, a, http.StatusOK)
}

func (h API) StoreTag(u *auth.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	tt, err := h.Handler.StoreTag(u, b, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to tag build"))
		return
	}
	webutil.JSON(w, tt, http.StatusCreated)
}

func (h API) ShowTag(u *auth.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	t, ok, err := h.Tags.Get(
		ctx,
		query.Where("build_id", "=", query.Arg(b.ID)),
		query.Where("name", "=", query.Arg(mux.Vars(r)["name"])),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get tag"))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	if err := user.Loader(h.DB).Load(ctx, "user_id", "id", t); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to load tag relation"))
		return
	}
	webutil.JSON(w, t, http.StatusOK)
}

func (h API) DestroyTag(u *auth.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	if err := h.Handler.DestroyTag(r.Context(), b, mux.Vars(r)["name"]); err != nil {
		if errors.Is(err, database.ErrNoRows) {
			h.NotFound(w, r)
			return
		}

		h.InternalServerError(w, r, errors.Wrap(err, "Failed to delete tag"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func RegisterAPI(a auth.Authenticator, srv *server.Server) {
	api := API{
		Handler: NewHandler(srv),
	}

	index := srv.Restrict(a, []string{"build:read"}, api.Index)
	store := srv.Restrict(a, []string{"build:write"}, api.Store)

	srv.Router.HandleFunc("/builds", index).Methods("GET")
	srv.Router.HandleFunc("/builds", store).Methods("POST")

	show := srv.Optional(a, api.Build(api.Show))
	destroy := srv.Restrict(a, []string{"build:delete"}, api.Build(api.Destroy))
	pin := srv.Restrict(a, []string{"build:write"}, api.Build(api.TogglePin))
	showJob := srv.Restrict(a, []string{"build:read"}, api.Build(api.ShowJob))
	download := srv.Restrict(a, []string{"build:read"}, api.Build(api.Download))
	storeTag := srv.Restrict(a, []string{"build:write"}, api.Build(api.StoreTag))
	showTag := srv.Restrict(a, []string{"build:read"}, api.Build(api.ShowTag))
	destroyTag := srv.Restrict(a, []string{"build:delete"}, api.Build(api.DestroyTag))

	sr := srv.Router.PathPrefix("/b/{username}/{build:[0-9]+}").Subrouter()
	sr.HandleFunc("", show).Methods("GET")
	sr.HandleFunc("", destroy).Methods("DELETE")
	sr.HandleFunc("/pin", pin).Methods("PATCH")
	sr.HandleFunc("/unpin", pin).Methods("PATCH")
	sr.HandleFunc("/objects", show).Methods("GET")
	sr.HandleFunc("/variables", show).Methods("GET")
	sr.HandleFunc("/keys", show).Methods("GET")
	sr.HandleFunc("/jobs", show).Methods("GET")
	sr.HandleFunc("/jobs/{name}", showJob).Methods("GET")
	sr.HandleFunc("/artifacts", show).Methods("GET")
	sr.HandleFunc("/artifacts/{name}", download).Methods("GET")
	sr.HandleFunc("/tags", show).Methods("GET")
	sr.HandleFunc("/tags", storeTag).Methods("POST")
	sr.HandleFunc("/tags/{name:.+}", showTag).Methods("GET")
	sr.HandleFunc("/tags/{name:.+}", destroyTag).Methods("DELETE")
}
