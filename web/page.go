package web

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/auth"
	"github.com/andrewpillar/thrall/template/build"
)

type Page struct {
	Handler
}

func NewPage(h Handler) Page {
	return Page{Handler: h}
}

func (h Page) Home(w http.ResponseWriter, r *http.Request) {
	u, err := h.UserFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if u.IsZero() {
		p := &auth.LoginPage{
			Errors: h.errors(w, r),
			Form:   h.form(w, r),
		}

		HTML(w, template.Render(p), http.StatusOK)
		return
	}

	builds, err := u.Builds()

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := model.LoadBuildRelations(builds); err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &build.IndexPage{
		Builds: builds,
	}

	d := template.NewDashboard(p, r.URL.Path)

	HTML(w, template.Render(d), http.StatusOK)
}
