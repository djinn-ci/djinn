package handler

import (
	"net/http"
	"strconv"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/mux"
)

// API is the handler for handling API requests made for build creation,
// submission, and retrieval.
type API struct {
	Build

	// Prefix is the part of the URL under which the API is being served, for
	// example "/api".
	Prefix string
}

// JobAPI is the handler for handling API requests made for working with the
// jobs within a build.
type JobAPI struct {
	Job

	// Prefix is the part of the URL under which the API is being served, for
	// example "/api".
	Prefix string
}

// TagAPI is the handler for handling API requests made for working with build
// tags.
type TagAPI struct {
	Tag

	// Prefix is the part of the URL under which the API is being served, for
	// example "/api".
	Prefix string
}

// Index serves the JSON encoded list of builds for the given request. If
// multiple pages of builds are returned then the database.Paginator is encoded
// in the Link response header.
func (h API) Index(w http.ResponseWriter, r *http.Request) {
	bb, paginator, err := h.IndexWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(bb))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, b := range bb {
		data = append(data, b.JSON(addr))
	}

	w.Header().Set("Link", web.EncodeToLink(paginator, r))
	webutil.JSON(w, data, http.StatusOK)
}

// Store stores and submits the build from the given request body. If any
// validation errors occur then these will be sent back in the JSON response.
// On success the build is sent in the JSON response.
func (h API) Store(w http.ResponseWriter, r *http.Request) {
	b, _, err := h.StoreModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.JSON(w, ferrs, http.StatusBadRequest)
			return
		}

		switch cause {
		case build.ErrDriver:
			errs := webutil.NewErrors()
			errs.Put("manifest", cause)

			webutil.JSON(w, errs, http.StatusBadRequest)
			return
		case namespace.ErrName:
			errs := webutil.NewErrors()
			errs.Put("manifest", cause)

			webutil.JSON(w, errs, http.StatusBadRequest)
			return
		case namespace.ErrPermission:
			webutil.JSON(w, map[string][]string{"namespace": {"Could not find namespace"}}, http.StatusBadRequest)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	builds := build.NewStoreWithHasher(h.DB, h.hasher)
	prd, _ := h.getDriverQueue(b.Manifest)
	addr := webutil.BaseAddress(r)

	if err := builds.Submit(r.Context(), prd, addr, b); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	webutil.JSON(w, b.JSON(addr+h.Prefix), http.StatusCreated)
}

// Show serves up the JSON response for the build in the given request. This
// serves different responses based on the base path of the request URL.
func (h API) Show(w http.ResponseWriter, r *http.Request) {
	b, err := h.ShowWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	base := webutil.BasePath(r.URL.Path)
	addr := webutil.BaseAddress(r) + h.Prefix

	switch base {
	case "objects":
		oo, err := h.objectsWithRelations(b)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		data := make([]interface{}, 0, len(oo))

		for _, o := range oo {
			json := o.JSON(addr)
			delete(json, "build")

			data = append(data, json)
		}
		webutil.JSON(w, data, http.StatusOK)
		return
	case "variables":
		vv, err := h.variablesWithRelations(b)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		data := make([]interface{}, 0, len(vv))

		for _, v := range vv {
			json := v.JSON(addr)
			delete(json, "build")

			data = append(data, json)
		}
		webutil.JSON(w, data, http.StatusOK)
		return
	case "keys":
		kk, err := build.NewKeyStore(h.DB, b).All()

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		data := make([]interface{}, 0, len(kk))

		for _, k := range kk {
			json := k.JSON(addr)
			delete(json, "build")

			data = append(data, json)
		}
		webutil.JSON(w, data, http.StatusOK)
		return
	}
	webutil.JSON(w, b.JSON(addr), http.StatusOK)
}

// Destroy kills the build in the given request.
func (h API) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.Kill(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Index serves the JSON encoded list of jobs within a build for the given
// request. This is not paginated.
func (h JobAPI) Index(w http.ResponseWriter, r *http.Request) {
	b, ok := build.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "Failed to get build from request context")
	}

	jj, err := h.IndexWithRelations(build.NewJobStore(h.DB, b), nil)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(jj))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, j := range jj {
		json := j.JSON(addr)
		delete(json, "build")

		data = append(data, json)
	}
	webutil.JSON(w, data, http.StatusOK)
}

// Show serves the JSON encoded build job for the given request.
func (h JobAPI) Show(w http.ResponseWriter, r *http.Request) {
	j, err := h.ShowWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if j.IsZero() {
		web.JSONError(w, "Not found", http.StatusNotFound)
		return
	}
	webutil.JSON(w, j.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

// Index serves the JSON encoded list of build tags for the build in the given
// request context.
func (h TagAPI) Index(w http.ResponseWriter, r *http.Request) {
	b, ok := build.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "Failed to get build from request context")
	}

	tt, err := build.NewTagStore(h.DB, b).All()

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	mm := database.ModelSlice(len(tt), build.TagModel(tt))

	err = h.Users.Load("id", database.MapKey("user_id", mm), database.Bind("user_id", "id", mm...))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(tt))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, t := range tt {
		json := t.JSON(addr)
		delete(json, "build")

		data = append(data, json)
	}
	webutil.JSON(w, data, http.StatusOK)
}

// Store adds the given tags in the request's body to the build in the given
// request context. This will serve a response to the JSON encoded list of
// tags that were added.
func (h TagAPI) Store(w http.ResponseWriter, r *http.Request) {
	tt, err := h.StoreModel(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(tt))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, t := range tt {
		json := t.JSON(addr)
		delete(json, "build")

		data = append(data, json)
	}
	webutil.JSON(w, data, http.StatusCreated)
}

// Show serves the JSON encoded response for an individual tag on the build in
// the given request context.
func (h TagAPI) Show(w http.ResponseWriter, r *http.Request) {
	b, ok := build.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "Failed to get build from request context")
	}

	if err := build.LoadRelations(h.loaders, b); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	id, _ := strconv.ParseInt(mux.Vars(r)["tag"], 10, 64)

	t, err := build.NewTagStore(h.DB, b).Get(query.Where("id", "=", query.Arg(id)))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if t.IsZero() {
		web.JSONError(w, "Not found", http.StatusNotFound)
		return
	}

	err = h.Users.Load("id", []interface{}{t.UserID}, database.Bind("user_id", "id", t))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	webutil.JSON(w, t.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

// Destroy removes the given tag from the build in the given request context.
// This serves no content as it's response.
func (h TagAPI) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
