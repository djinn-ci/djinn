package handler

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/variable"
	variabletemplate "github.com/andrewpillar/thrall/variable/template"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"
)

type UI struct {
	Variable
}

func (h UI) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u := h.User(r)

	vv, paginator, err := h.IndexWithRelations(variable.NewStore(h.DB, u), r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	csrfField := string(csrf.TemplateField(r))

	p := &variabletemplate.Index{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      csrfField,
		Search:    r.URL.Query().Get("search"),
		Paginator: paginator,
		Variables: vv,
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrfField := string(csrf.TemplateField(r))

	p := &variabletemplate.Create{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: h.FormErrors(sess),
			Fields: h.FormFields(sess),
		},
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Variable) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	v, err := h.StoreModel(r, sess)

	if err != nil {
		cause := errors.Cause(err)

		if _, ok := cause.(form.Errors); ok {
			h.RedirectBack(w, r)
			return
		}

		if cause == namespace.ErrPermission {
			sess.AddFlash(template.Danger("Failed to create variable: could not add to namespace"), "alert")
		} else {
			sess.AddFlash(template.Danger("Failed to create variable"), "alert")
			log.Error.Println(errors.Err(err))
		}
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Variable has been added: "+v.Key), "alert")
	h.Redirect(w, r, "/variables")
}

func (h Variable) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	v := h.Model(r)

	if err := h.Variables.Delete(v); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to delete variable"), "alert")
		h.RedirectBack(w, r)
		return
	}
	sess.AddFlash(template.Success("Variable has been deleted: "+v.Key), "alert")
	h.RedirectBack(w, r)
}
