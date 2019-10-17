package ui

import (
	"net/http"
	"os"
	"regexp"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/file"
	"github.com/andrewpillar/thrall/template/object"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/web/core"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type Object struct {
	Core core.Object

	Build     Build
}

func (h Object) indexPage(objects model.ObjectStore, r *http.Request) (object.IndexPage, error) {
	u := h.Core.User(r)

	oo, paginator, err := h.Core.Index(objects, r)

	if err != nil {
		return object.IndexPage{}, errors.Err(err)
	}

	p := object.IndexPage{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      string(csrf.TemplateField(r)),
		Paginator: paginator,
		Objects:   oo,
		Search:    r.URL.Query().Get("search"),
	}

	return p, nil
}

func (h Object) Index(w http.ResponseWriter, r *http.Request) {
	u := h.Core.User(r)

	p, err := h.indexPage(u.ObjectStore(), r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	d := template.NewDashboard(&p, r.URL, h.Core.Alert(w, r), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Object) Show(w http.ResponseWriter, r *http.Request) {
	u := h.Core.User(r)

	o, err := h.Core.Show(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	bp, err := h.Build.indexPage(u.BuildStore(), r, query.WhereQuery("id", "IN",
		query.Select(
			query.Columns("build_id"),
			query.From("build_objects"),
			query.Where("object_id", "=", o.ID),
		),
	))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &object.ShowPage{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:   string(csrf.TemplateField(r)),
		Object: o,
		Index:  bp,
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(w, r), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Object) Create(w http.ResponseWriter, r *http.Request) {
	p := &file.CreatePage{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Core.Errors(w, r),
			Fields: h.Core.Form(w, r),
		},
		Name:   "object",
		Action: "/objects",
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(w, r), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Object) Store(w http.ResponseWriter, r *http.Request) {
	o, err := h.Core.Store(w, r)

	if err != nil {
		cause := errors.Cause(err)

		switch cause {
		case core.ErrValidationFailed:
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		case core.ErrAccessDenied:
			h.Core.FlashAlert(w, r, template.Danger("Failed to create object: could not add to namespace"))
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		default:
			log.Error.Println(errors.Err(err))

			h.Core.FlashAlert(w, r, template.Danger("Failed to create object: " + cause.Error()))
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}
	}

	h.Core.FlashAlert(w, r, template.Success("Object has been added: " + o.Name))

	http.Redirect(w, r, "/objects", http.StatusSeeOther)
}

func (h Object) Download(w http.ResponseWriter, r *http.Request) {
	o := h.Core.Object(r)

	vars := mux.Vars(r)

	if o.Name != vars["name"] {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	f, err := h.Core.FileStore.Open(o.Hash)

	if err != nil {
		if os.IsNotExist(errors.Cause(err)) {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	defer f.Close()

	http.ServeContent(w, r, o.Name, o.UpdatedAt, f)
}

func (h Object) Destroy(w http.ResponseWriter, r *http.Request) {
	o := h.Core.Object(r)

	if err := h.Core.Destroy(r); err != nil {
		cause := errors.Cause(err)

		if !os.IsNotExist(cause) {
			log.Error.Println(errors.Err(err))
		}

		h.Core.FlashAlert(w, r, template.Danger("Failed to delete object: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.Core.FlashAlert(w, r, template.Success("Object has been deleted: " + o.Name))

	ref := r.Header.Get("Referer")

	if matched, err := regexp.Match("/objects/[0-9]+", []byte(ref)); err == nil || matched {
		http.Redirect(w, r, "/objects", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
