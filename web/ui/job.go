package ui

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/job"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/mux"
)

type Job struct {
	web.Handler
}

func (h Job) Show(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)

	buildId, _ := strconv.ParseInt(vars["build"], 10, 64)

	b, err := u.BuildStore().Find(buildId)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	jobId, _ := strconv.ParseInt(vars["job"], 10, 64)

	j, err := b.JobStore().Show(jobId)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if j.IsZero() {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	if filepath.Base(r.URL.Path) == "raw" {
		web.Text(w, j.Output.String, http.StatusOK)
		return
	}

	p := &job.ShowPage{
		BasePage: template.BasePage{
			URI: r.URL.Path,
		},
		Job: j,
	}

	d := template.NewDashboard(p, r.URL.Path, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}
