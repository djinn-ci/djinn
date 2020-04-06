package handler

import (
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/andrewpillar/thrall/build"
	buildtemplate "github.com/andrewpillar/thrall/build/template"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/object"
	objecttemplate "github.com/andrewpillar/thrall/object/template"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type UI struct {
	Object
}

func (h UI) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u := h.User(r)

	oo, paginator, err := h.IndexWithRelations(object.NewStore(h.DB, u), r)

	if err != nil {
		log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	csrfField := string(csrf.TemplateField(r))

	p := &objecttemplate.Index{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      csrfField,
		Paginator: paginator,
		Objects:   oo,
		Search:    r.URL.Query().Get("search"),
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrfField := string(csrf.TemplateField(r))

	f := template.Form{
		CSRF:   csrfField,
		Errors: h.FormErrors(sess),
		Fields: h.FormFields(sess),
	}

	p := &objecttemplate.Create{
		Form:     f,
		FileForm: &template.FileForm{
			Form: f,
		},
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	o, err := h.StoreModel(w, r, sess)

	if err != nil {
		cause := errors.Cause(err)

		if _, ok := cause.(form.Errors); !ok {
			log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to create object"), "alert")
		}
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Object has been added: "+o.Name), "alert")
	h.Redirect(w, r, "/objects")
}

func (h UI) Show(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")

	if parts[len(parts)-2] == "download" {
		o := h.Model(r)

		if o.Name != mux.Vars(r)["name"] {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		f, err := h.FileStore.Open(o.Hash)

		if err != nil {
			if os.IsNotExist(errors.Cause(err)) {
				web.HTMLError(w, "Not found", http.StatusNotFound)
				return
			}
			log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		defer f.Close()

		http.ServeContent(w, r, o.Name, o.CreatedAt, f)
		return
	}

	sess, save := h.Session(r)

	u := h.User(r)

	o, err := h.ShowWithRelations(r)

	if err != nil {
		log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	selectq := build.SelectObject("build_id", query.Where("object_id", "=", o.ID))

	bb, paginator, err := h.Builds.Index(r, query.WhereQuery("id", "IN", selectq))

	if err != nil {
		log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()

	csrfField := string(csrf.TemplateField(r))

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}
	p := &objecttemplate.Show{
		BasePage: bp,
		CSRF:     csrfField,
		Object:   o,
		Section:  &buildtemplate.Index{
			BasePage:  bp,
			Paginator: paginator,
			Builds:    bb,
			Search:    q.Get("search"),
			Status:    q.Get("status"),
			Tag:       q.Get("tag"),
		},
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	alert := template.Success("Object has been deleted")

	if err := h.Delete(r); err != nil {
		if !os.IsNotExist(errors.Cause(err)) {
			log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
			alert = template.Danger("Failed to delete object")
		}
	}

	sess.AddFlash(alert, "alert")

	ref := r.Header.Get("Referer")

	if matched, _ := regexp.Match("/objects/[0-9]+", []byte(ref)); matched {
		h.Redirect(w, r, "/objects")
		return
	}
	h.RedirectBack(w, r)
}
