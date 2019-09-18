package ui

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/job"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/mux"
)

type Job struct {
	web.Handler
}

func (h Job) Build(r *http.Request) *model.Build {
	val := r.Context().Value("build")

	b, _ := val.(*model.Build)

	return b
}

func (h Job) Show(w http.ResponseWriter, r *http.Request) {
	b := h.Build(r)

	vars := mux.Vars(r)

	jobId, _ := strconv.ParseInt(vars["job"], 10, 64)

	j, err := b.JobStore().Show(jobId)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if filepath.Base(r.URL.Path) == "raw" {
		web.Text(w, j.Output.String, http.StatusOK)
		return
	}

	p := &job.ShowPage{
		BasePage: template.BasePage{
			URL: r.URL,
		},
		Job: j,
	}

	d := template.NewDashboard(p, r.URL, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}
