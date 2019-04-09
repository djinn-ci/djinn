package web

import (
	"net/http"
	"strconv"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/job"

	"github.com/gorilla/mux"
)

type Job struct {
	Handler
}

func NewJob(h Handler) Job {
	return Job{Handler: h}
}

func (h Job) Show(w http.ResponseWriter, r *http.Request) {
	u, err := h.userFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)

	buildId, err := strconv.ParseInt(vars["build"], 10, 64)

	if err != nil {
		HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	b, err := u.FindBuild(buildId)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if b.IsZero() {
		HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	jobId, err := strconv.ParseInt(vars["job"], 10, 64)

	if err != nil {
		HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	j, err := b.FindJob(jobId)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if j.IsZero() {
		HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	if err := j.LoadRelations(); err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	j.Stage.Build = b

	p := &job.ShowPage{
		Page: &template.Page{
			URI: r.URL.Path,
		},
		Job: j,
	}

	d := template.NewDashboard(p, r.URL.Path)

	HTML(w, template.Render(d), http.StatusOK)
}
