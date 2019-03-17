package web

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/build"

	"github.com/gorilla/mux"
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

func (h Build) Store(w http.ResponseWriter, r *http.Request) {
	u, err := h.userFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	f := &form.Build{}

	if err := h.handleRequestData(f, w, r); err != nil {
		return
	}

	b := model.Build{
		UserID:   u.ID,
		Manifest: f.Manifest,
	}

	if f.Namespace != "" {
		n, err := u.FindOrCreateNamespace(f.Namespace)

		if err != nil {
			log.Error.Println(errors.Err(err))
			HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		b.NamespaceID = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}
	}

	if err := b.Create(); err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	tags := make([]model.BuildTag, len(f.Tags), len(f.Tags))

	for i, name := range f.Tags {
		if name == "" {
			continue
		}

		tags[i] = model.BuildTag{
			UserID:  u.ID,
			BuildID: b.ID,
			Name:    strings.TrimSpace(name),
		}

		if err := tags[i].Create(); err != nil {
			log.Error.Println(errors.Err(err))
			HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h Build) Show(w http.ResponseWriter, r *http.Request) {
	u, err := h.userFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)

	id, err := strconv.ParseInt(vars["build"], 10, 64)

	if err != nil {
		HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	b, err := u.FindBuild(id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if b.IsZero() {
		HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	p := &build.ShowPage{
		Page: &template.Page{
			URI: r.URL.Path,
		},
		Build: b,
	}

	d := template.NewDashboard(p, r.URL.Path)

	HTML(w, template.Render(d), http.StatusOK)
}
