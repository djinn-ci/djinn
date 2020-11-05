package handler

import (
	"net/http"
	"strconv"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
)

type Artifact struct {
	web.Handler

	loaders *database.Loaders

	Prefix string
}

func NewArtifact(h web.Handler) Artifact {
	loaders := database.NewLoaders()
	loaders.Put("user", user.NewStore(h.DB))
	loaders.Put("namespace", namespace.NewStore(h.DB))
	loaders.Put("build_tag", build.NewTagStore(h.DB))
	loaders.Put("build_trigger", build.NewTriggerStore(h.DB))

	return Artifact{
		Handler: h,
		loaders: loaders,
	}
}

// Index serves the JSON encoded list of artifacts for the build in the given
// request context.
func (h Artifact) Index(w http.ResponseWriter, r *http.Request) {
	b, ok := build.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "Failed to get build from request context")
	}

	aa, err := build.NewArtifactStore(h.DB, b).All(database.Search("name", r.URL.Query().Get("search")))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
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

// Show serves the JSON encoded data of the given build artifact for the build
// in the given request context.
func (h Artifact) Show(w http.ResponseWriter, r *http.Request) {
	b, ok := build.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "Failed to get build from request context")
	}

	if err := build.LoadRelations(h.loaders, b); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	id, _ := strconv.ParseInt(mux.Vars(r)["artifact"], 10, 64)

	a, err := build.NewArtifactStore(h.DB, b).Get(query.Where("id", "=", id))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if a.IsZero() {
		web.JSONError(w, "Not found", http.StatusNotFound)
		return
	}
	web.JSON(w, a.JSON(web.BaseAddress(r)+h.Prefix), http.StatusOK)
}
