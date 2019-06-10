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

func NewJob(h web.Handler) Job {
	return Job{
		Handler: h,
	}
}

func (h Job) Show(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)

	buildId, err := strconv.ParseInt(vars["build"], 10, 64)

	if err != nil {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	b, err := u.BuildStore().Find(buildId)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if b.IsZero() {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	jobId, err := strconv.ParseInt(vars["job"], 10, 64)

	if err != nil {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	j, err := b.JobShow(jobId)

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

	if err := j.LoadArtifacts(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := j.LoadBuild(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := j.LoadDependencies(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &job.ShowPage{
		Page: template.Page{
			URI: r.URL.Path,
		},
		Job:        j,
		ShowOutput: filepath.Base(r.URL.Path) == "output",
	}

	d := template.NewDashboard(p, r.URL.Path)

	web.HTML(w, template.Render(d), http.StatusOK)
}
