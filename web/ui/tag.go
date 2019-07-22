package ui

import (
	"net/http"
	"strconv"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/build"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type Tag struct {
	web.Handler
}

func (h Tag) Index(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)

	id, _ := strconv.ParseInt(vars["build"], 10, 64)

	b, err := u.BuildStore().Find(id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	tt, err := b.TagStore().Index()

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &build.TagIndexPage{
		ShowPage: build.ShowPage{
			Page: template.Page{
				URI: r.URL.Path,
			},
			Build: b,
		},
		CSRF: csrf.TemplateField(r),
		Tags: tt,
	}

	d := template.NewDashboard(p, r.URL.Path, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Tag) Store(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)

	id, _ := strconv.ParseInt(vars["build"], 10, 64)

	b, err := u.BuildStore().Find(id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	f := &form.Tag{}

	if err := form.Unmarshal(f, r); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if len(f.Tags) == 0 {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	tags := b.TagStore()

	for _, tag := range f.Tags {
		t := tags.New()
		t.UserID = u.ID
		t.Name = tag

		if err := t.Create(); err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h Tag) Destroy(w http.ResponseWriter, r *http.Request) {
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

	tagId, err := strconv.ParseInt(vars["tag"], 10, 64)

	if err != nil {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	t, err := b.TagStore().Find(tagId)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if t.IsZero() {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	if err := t.Destroy(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	h.FlashAlert(w, r, template.Success("Tag has been deleted:" + t.Name))

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
