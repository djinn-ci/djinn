package ui

import (
	"net/http"
	"path/filepath"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/job"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/web/core"
)

type Job struct {
	Core core.Job
}

func (h Job) Show(w http.ResponseWriter, r *http.Request) {
	j, err := h.Core.Show(r)

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

	d := template.NewDashboard(p, r.URL, h.Core.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}
