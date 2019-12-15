package ui

import (
	"net/http"
	"os"
	"strconv"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
)

type Artifact struct {
	web.Handler

	FileStore filestore.FileStore
}

func (h Artifact) Show(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	vars := mux.Vars(r)

	buildId, _ := strconv.ParseInt(vars["build"], 10, 64)

	artifactId, _ := strconv.ParseInt(vars["artifact"], 10, 64)

	b, err := u.BuildStore().Get(query.Where("id", "=", buildId))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	a, err := b.ArtifactStore().Get(query.Where("id", "=", artifactId))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if a.IsZero() || a.Name != vars["name"] {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	f, err := h.FileStore.Open(a.Hash)

	if err != nil {
		if os.IsNotExist(errors.Cause(err)) {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	defer f.Close()

	http.ServeContent(w, r, a.Name, a.UpdatedAt, f)
}
