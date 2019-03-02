package web

import (
	"net/http"

	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/build"
)

type Build struct {
	Handler
}

func NewBuild(h Handler) Build {
	return Build{Handler: h}
}

func (h Build) Create(w http.ResponseWriter, r *http.Request) {
	p := &build.CreatePage{
		Errors: h.errors(w, r),
		Form:   h.form(w, r),
	}

	d := template.NewDashboard(p, r.URL.RequestURI())

	HTML(w, template.Render(d), http.StatusOK)
}
