package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/runner"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
)

type API struct {
	Build

	Prefix   string
	Tag      TagAPI
}

type ArtifactAPI struct {
	web.Handler

	Prefix   string
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
	u := h.User(r)

	bb, paginator, err := h.IndexWithRelations(build.NewStore(h.DB, u), r.URL.Query())

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
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
	defer r.Body.Close()

	b, err := h.StoreModel(r, nil)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.JSON(w, ferrs, http.StatusBadRequest)
			return
		}

		switch cause {
		case build.ErrDriver:
			web.JSON(w, map[string][]string{"manifest":[]string{cause.Error()}}, http.StatusBadRequest)
			return
		case namespace.ErrPermission:
			web.JSONError(w, "Could not add to namespace", http.StatusBadRequest)
			return
		default:
			log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	if err := h.Submit(b, h.Queues[b.Manifest.Driver["type"]]); err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	web.JSON(w, b.JSON(web.BaseAddress(r) + h.Prefix), http.StatusCreated)
}

func (h API) Show(w http.ResponseWriter, r *http.Request) {
	b, err := h.ShowWithRelations(r)

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	base := filepath.Base(r.URL.Path)
	addr := web.BaseAddress(r) + h.Prefix

	switch base {
	case "objects":
		oo, err := h.objectsWithRelations(b)

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(err))
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
			log.Error.Println(r.Method, r.URL, errors.Err(err))
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
			log.Error.Println(r.Method, r.URL, errors.Err(err))
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

func (h API) Kill(w http.ResponseWriter, r *http.Request) {
	b := Model(r)

	if b.Status != runner.Running {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if _, err := h.Client.Publish(fmt.Sprintf("kill-%v", b.ID), b.Secret.String).Result(); err != nil {
		log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h JobAPI) Index(w http.ResponseWriter, r *http.Request) {
	b := Model(r)

	jj, err := h.IndexWithRelations(build.NewJobStore(h.DB, b), nil)

	if err != nil {
		log.Error.Println(errors.Err(err))
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
		log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	web.JSON(w, j.JSON(web.BaseAddress(r) + h.Prefix), http.StatusOK)
}

func (h ArtifactAPI) Index(w http.ResponseWriter, r *http.Request) {
	b := Model(r)

	aa, err := build.NewArtifactStore(h.DB, b).All(model.Search("name", r.URL.Query().Get("search")))

	if err != nil {
		log.Error.Println(errors.Err(err))
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
	b := Model(r)

	err := build.NewTriggerStore(h.DB).Load("build_id", []interface{}{b.ID}, model.Bind("build_id", "id", b))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	id, _ := strconv.ParseInt(mux.Vars(r)["artifact"], 10, 64)

	a, err := build.NewArtifactStore(h.DB, b).Get(query.Where("id", "=", id))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	web.JSON(w, a.JSON(web.BaseAddress(r) + h.Prefix), http.StatusOK)
}

func (h TagAPI) Index(w http.ResponseWriter, r *http.Request) {
	b := Model(r)

	tt, err := build.NewTagStore(h.DB, b).All()

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	mm := model.Slice(len(tt), build.TagModel(tt))

	err = h.Users.Load("id", model.MapKey("user_id", mm), model.Bind("user_id", "id", mm...))

	if err != nil {
		log.Error.Println(errors.Err(err))
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
	defer r.Body.Close()

	u := h.User(r)
	b := Model(r)

	tags := []string{}

	if err := json.NewDecoder(r.Body).Decode(&tags); err != nil {
		web.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	store := build.NewTagStore(h.DB, b)

	tt := make([]*build.Tag, 0, len(tags))

	for _, name := range tags {
		t := store.New()
		t.UserID = u.ID
		t.Name = name

		tt = append(tt, t)
	}

	if err := store.Create(tt...); err != nil {
		log.Error.Println(errors.Err(err))
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
	b := Model(r)

	id, _ := strconv.ParseInt(mux.Vars(r)["tag"], 10, 64)

	t, err := build.NewTagStore(h.DB, b).Get(query.Where("id", "=", id))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if t.IsZero() {
		web.JSONError(w, "Not found", http.StatusNotFound)
	}
	web.JSON(w, t.JSON(web.BaseAddress(r) + h.Prefix), http.StatusOK)
}

func (h TagAPI) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.Delete(r); err != nil {
		log.Error.Println(errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
