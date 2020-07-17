package handler

import (
	"net/http"
	"strconv"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
)

type API struct {
	Build

	Prefix string
}

type ArtifactAPI struct {
	web.Handler

	Prefix string
}

type JobAPI struct {
	Job

	Prefix string
}

type TagAPI struct {
	Tag

	Prefix string
}

func (h API) Index(w http.ResponseWriter, r *http.Request) {
	bb, paginator, err := h.IndexWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(bb))
	addr := web.BaseAddress(r) + h.Prefix

	for _, b := range bb {
		data = append(data, b.JSON(addr))
	}

	w.Header().Set("Link", web.EncodeToLink(paginator, r))
	web.JSON(w, data, http.StatusOK)
}

func (h API) Store(w http.ResponseWriter, r *http.Request) {
	b, _, err := h.StoreModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.JSON(w, ferrs, http.StatusBadRequest)
			return
		}

		switch cause {
		case build.ErrDriver:
			errs := form.NewErrors()
			errs.Put("manifest", cause)

			web.JSON(w, errs, http.StatusBadRequest)
			return
		case namespace.ErrName:
			errs := form.NewErrors()
			errs.Put("manifest", cause)

			web.JSON(w, errs, http.StatusBadRequest)
			return
		case namespace.ErrPermission:
			web.JSONError(w, "Unprocessable entity", http.StatusUnprocessableEntity)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	if err := build.NewStoreWithHasher(h.DB, h.Hasher).Submit(h.Queues[b.Manifest.Driver["type"]], b); err != nil {
		h.Log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	web.JSON(w, b.JSON(web.BaseAddress(r) + h.Prefix), http.StatusCreated)
}

func (h API) Show(w http.ResponseWriter, r *http.Request) {
	b, err := h.ShowWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	base := web.BasePath(r.URL.Path)
	addr := web.BaseAddress(r) + h.Prefix

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
		web.JSON(w, data, http.StatusOK)
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
		web.JSON(w, data, http.StatusOK)
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
		web.JSON(w, data, http.StatusOK)
		return
	}
	web.JSON(w, b.JSON(addr), http.StatusOK)
}

func (h API) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.Kill(r); err != nil {
		h.Log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h JobAPI) Index(w http.ResponseWriter, r *http.Request) {
	b, ok := build.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "Failed to get build from request context")
	}

	jj, err := h.IndexWithRelations(build.NewJobStore(h.DB, b), nil)

	if err != nil {
		h.Log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(jj))
	addr := web.BaseAddress(r) + h.Prefix

	for _, j := range jj {
		json := j.JSON(addr)
		delete(json, "build")

		data = append(data, json)
	}
	web.JSON(w, data, http.StatusOK)
}

func (h JobAPI) Show(w http.ResponseWriter, r *http.Request) {
	j, err := h.ShowWithRelations(r)

	if err != nil {
		h.Log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if j.IsZero() {
		web.JSONError(w, "Not found", http.StatusNotFound)
		return
	}
	web.JSON(w, j.JSON(web.BaseAddress(r) + h.Prefix), http.StatusOK)
}

func (h ArtifactAPI) Index(w http.ResponseWriter, r *http.Request) {
	b, ok := build.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "Failed to get build from request context")
	}

	aa, err := build.NewArtifactStore(h.DB, b).All(database.Search("name", r.URL.Query().Get("search")))

	if err != nil {
		h.Log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(aa))
	addr := web.BaseAddress(r) + h.Prefix

	for _, a := range aa {
		json := a.JSON(addr)
		delete(json, "build")

		data = append(data, json)
	}
	web.JSON(w, data, http.StatusOK)
}

func (h ArtifactAPI) Show(w http.ResponseWriter, r *http.Request) {
	b, ok := build.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "Failed to get build from request context")
	}

	err := build.NewTriggerStore(h.DB).Load("build_id", []interface{}{b.ID}, database.Bind("build_id", "id", b))

	if err != nil {
		h.Log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	id, _ := strconv.ParseInt(mux.Vars(r)["artifact"], 10, 64)

	a, err := build.NewArtifactStore(h.DB, b).Get(query.Where("id", "=", id))

	if err != nil {
		h.Log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if a.IsZero() {
		web.JSONError(w, "Not found", http.StatusNotFound)
		return
	}
	web.JSON(w, a.JSON(web.BaseAddress(r) + h.Prefix), http.StatusOK)
}

func (h TagAPI) Index(w http.ResponseWriter, r *http.Request) {
	b, ok := build.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "Failed to get build from request context")
	}

	tt, err := build.NewTagStore(h.DB, b).All()

	if err != nil {
		h.Log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	mm := database.ModelSlice(len(tt), build.TagModel(tt))

	err = h.Users.Load("id", database.MapKey("user_id", mm), database.Bind("user_id", "id", mm...))

	if err != nil {
		h.Log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(tt))
	addr := web.BaseAddress(r) + h.Prefix

	for _, t := range tt {
		json := t.JSON(addr)
		delete(json, "build")

		data = append(data, json)
	}
	web.JSON(w, data, http.StatusOK)
}

func (h TagAPI) Store(w http.ResponseWriter, r *http.Request) {
	tt, err := h.StoreModel(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(tt))
	addr := web.BaseAddress(r) + h.Prefix

	for _, t := range tt {
		json := t.JSON(addr)
		delete(json, "build")

		data = append(data, json)
	}
	web.JSON(w, data, http.StatusCreated)
}

func (h TagAPI) Show(w http.ResponseWriter, r *http.Request) {
	b, ok := build.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "Failed to get build from request context")
	}

	id, _ := strconv.ParseInt(mux.Vars(r)["tag"], 10, 64)

	t, err := build.NewTagStore(h.DB, b).Get(query.Where("id", "=", id))

	if err != nil {
		h.Log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if t.IsZero() {
		web.JSONError(w, "Not found", http.StatusNotFound)
	}
	web.JSON(w, t.JSON(web.BaseAddress(r) + h.Prefix), http.StatusOK)
}

func (h TagAPI) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
