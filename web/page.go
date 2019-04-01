package web

import (
	"net/http"
	"strings"
	"time"

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
	u, err := h.userFromRequest(r)

	if err != nil {
		cause := errors.Cause(err)

		if strings.Contains(cause.Error(), "expired timestamp") {
			cookie := &http.Cookie{
				Name:     "user",
				HttpOnly: true,
				Path:     "/",
				Expires:  time.Unix(0, 0),
			}

			http.SetCookie(w, cookie)
		} else {
			log.Error.Println(errors.Err(err))
			HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	if u.IsZero() {
		p := &auth.LoginPage{
			Errors: h.errors(w, r),
			Form:   h.form(w, r),
		}

		HTML(w, template.Render(p), http.StatusOK)
		return
	}

	var builds []*model.Build

	status := r.URL.Query().Get("status")

	if status != "" {
		builds, err = u.BuildsByStatus(status)
	} else {
		builds, err = u.Builds()
	}

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
		Page: &template.Page{
			URI: r.URL.Path,
		},
		Builds: builds,
		Status: status,
	}

	d := template.NewDashboard(p, r.URL.Path)

	HTML(w, template.Render(d), http.StatusOK)
}
