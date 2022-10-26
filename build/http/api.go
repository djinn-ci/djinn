package http

import (
	"net/http"

	"djinn-ci.com/build"
	"djinn-ci.com/database"
	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/mux"
)

type API struct {
	*Handler

	// Prefix is the endpoint underwhich the API is being served.
	Prefix string
}

func (h API) Index(u *user.User, w http.ResponseWriter, r *http.Request) {
	bb, paginator, err := h.IndexWithRelations(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	data := make([]map[string]interface{}, 0, len(bb))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, b := range bb {
		data = append(data, b.JSON(addr))
	}

	w.Header().Set("Link", paginator.EncodeToLink(r.URL))
	webutil.JSON(w, data, http.StatusOK)
}

func (h API) Store(u *user.User, w http.ResponseWriter, r *http.Request) {
	b, _, err := h.StoreModel(u, r)

	if err != nil {
		cause := errors.Cause(err)

		errs := webutil.NewValidationErrors()

		switch err := cause.(type) {
		case webutil.ValidationErrors:
			if errs, ok := err["fatal"]; ok {
				h.InternalServerError(w, r, errors.Slice(errs))
				return
			}
			webutil.JSON(w, err, http.StatusBadRequest)
		case *namespace.PathError:
			webutil.JSON(w, map[string][]string{"namespace": {err.Error()}}, http.StatusBadRequest)
		case *driver.Error:
			errs.Add("manifest", err)
			webutil.JSON(w, errs, http.StatusBadRequest)
		default:
			h.InternalServerError(w, r, errors.Err(err))
		}
		return
	}
	webutil.JSON(w, b.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

func (h API) Show(u *user.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	if err := build.LoadRelations(h.DB, b); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if err := build.LoadNamespaces(h.DB, b); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	base := webutil.BasePath(r.URL.Path)
	addr := webutil.BaseAddress(r) + h.Prefix

	switch base {
	case "objects":
		oo, err := h.objectsWithRelations(b)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		data := make([]map[string]interface{}, 0, len(oo))

		for _, o := range oo {
			o.Build = b
			data = append(data, o.JSON(addr))
		}
		webutil.JSON(w, data, http.StatusOK)
		return
	case "artifacts":
		aa, err := h.Artifacts.All(
			query.Where("build_id", "=", query.Arg(b.ID)),
			database.Search("name", r.URL.Query().Get("search")),
			query.OrderAsc("name"),
		)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		data := make([]map[string]interface{}, 0, len(aa))

		for _, a := range aa {
			a.Build = b
			data = append(data, a.JSON(addr))
		}
		webutil.JSON(w, data, http.StatusOK)
		return
	case "variables":
		vv, err := h.variablesWithRelations(b)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		data := make([]map[string]interface{}, 0, len(vv))

		for _, v := range vv {
			v.Build = b
			data = append(data, v.JSON(addr))
		}
		webutil.JSON(w, data, http.StatusOK)
		return
	case "keys":
		kk, err := h.Keys.All(query.Where("build_id", "=", query.Arg(b.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		data := make([]map[string]interface{}, 0, len(kk))

		for _, k := range kk {
			k.Build = b
			data = append(data, k.JSON(addr))
		}
		webutil.JSON(w, data, http.StatusOK)
		return
	case "jobs":
		jj, err := h.Jobs.Index(r.URL.Query(), query.Where("build_id", "=", query.Arg(b.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if err := build.LoadJobRelations(h.DB, jj...); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		data := make([]map[string]interface{}, 0, len(jj))

		for _, j := range jj {
			j.Build = b
			data = append(data, j.JSON(addr))
		}
		webutil.JSON(w, data, http.StatusOK)
		return
	case "tags":
		tt, err := h.Tags.All(query.Where("build_id", "=", query.Arg(b.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		data := make([]map[string]interface{}, 0, len(tt))

		for _, t := range tt {
			t.Build = b
			data = append(data, t.JSON(addr))
		}
		webutil.JSON(w, data, http.StatusOK)
		return
	}
	webutil.JSON(w, b.JSON(addr), http.StatusOK)
}

func (h API) ShowJob(u *user.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	j, ok, err := h.Jobs.Get(
		query.Where("build_id", "=", query.Arg(b.ID)),
		query.Where("name", "=", query.Arg(mux.Vars(r)["name"])),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	b.User, _, err = h.Users.Get(query.Where("id", "=", query.Arg(b.UserID)))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	j.Build = b

	if err := build.LoadJobRelations(h.DB, j); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	webutil.JSON(w, j.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

func (h API) Destroy(u *user.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	if err := b.Kill(h.Redis); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h API) TogglePin(u *user.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	base := webutil.BasePath(r.URL.Path)

	var err error

	switch base {
	case "pin":
		err = b.Pin(h.DB)
	case "unpin":
		err = b.Unpin(h.DB)
	}

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	h.Queues.Produce(r.Context(), "events", &build.PinEvent{Build: b})

	webutil.JSON(w, b.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

func (h API) Download(u *user.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	a, ok, err := h.Artifacts.Get(
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
		store, err := h.Artifacts.Partition(b.UserID)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		rec, err := store.Open(a.Hash)

		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				h.NotFound(w, r)
				return
			}

			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		defer rec.Close()
		http.ServeContent(w, r, a.Name, a.CreatedAt, rec)
		return
	}
	webutil.JSON(w, a.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

func (h API) StoreTag(u *user.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	tt, err := h.StoreTagModel(u, b, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	data := make([]map[string]interface{}, 0, len(tt))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, t := range tt {
		data = append(data, t.JSON(addr))
	}
	webutil.JSON(w, data, http.StatusCreated)
}

func (h API) ShowTag(u *user.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	t, ok, err := h.Tags.Get(
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

	if err := h.Users.Load("user_id", "id", t); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	webutil.JSON(w, t.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

func (h API) DestroyTag(u *user.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteTagModel(b, mux.Vars(r)); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.NotFound(w, r)
			return
		}

		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func RegisterAPI(prefix string, srv *server.Server) {
	user := userhttp.NewHandler(srv)

	api := API{
		Handler: NewHandler(srv),
		Prefix:  prefix,
	}

	auth := srv.Router.PathPrefix("/builds").Subrouter()
	auth.HandleFunc("", user.WithUser(api.Index)).Methods("GET", "HEAD")
	auth.HandleFunc("", user.WithUser(api.Store)).Methods("POST")

	sr := srv.Router.PathPrefix("/b/{username}/{build:[0-9]+}").Subrouter()
	sr.HandleFunc("", user.WithUser(api.WithBuild(api.Show))).Methods("GET")
	sr.HandleFunc("", user.WithUser(api.WithBuild(api.Destroy))).Methods("DELETE")
	sr.HandleFunc("/pin", user.WithUser(api.WithBuild(api.TogglePin))).Methods("PATCH")
	sr.HandleFunc("/unpin", user.WithUser(api.WithBuild(api.TogglePin))).Methods("PATCH")
	sr.HandleFunc("/objects", user.WithUser(api.WithBuild(api.Show))).Methods("GET")
	sr.HandleFunc("/variables", user.WithUser(api.WithBuild(api.Show))).Methods("GET")
	sr.HandleFunc("/keys", user.WithUser(api.WithBuild(api.Show))).Methods("GET")
	sr.HandleFunc("/jobs", user.WithUser(api.WithBuild(api.Show))).Methods("GET")
	sr.HandleFunc("/jobs/{name}", user.WithUser(api.WithBuild(api.ShowJob))).Methods("GET")
	sr.HandleFunc("/artifacts", user.WithUser(api.WithBuild(api.Show))).Methods("GET")
	sr.HandleFunc("/artifacts/{name}", user.WithUser(api.WithBuild(api.Download))).Methods("GET")
	sr.HandleFunc("/tags", user.WithUser(api.WithBuild(api.Show))).Methods("GET")
	sr.HandleFunc("/tags", user.WithUser(api.WithBuild(api.StoreTag))).Methods("POST")
	sr.HandleFunc("/tags/{name:.+}", user.WithUser(api.WithBuild(api.ShowTag))).Methods("GET")
	sr.HandleFunc("/tags/{name:.+}", user.WithUser(api.WithBuild(api.DestroyTag))).Methods("DELETE")
}
