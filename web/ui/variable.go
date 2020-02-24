package ui

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/variable"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/web/core"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
)

type Variable struct {
	Core core.Variable
}

func (h Variable) indexPage(variables model.VariableStore, r *http.Request, opts ...query.Option) (variable.IndexPage, error) {
	u := h.Core.User(r)

	vv, paginator, err := h.Core.Index(variables, r, opts...)

	if err != nil {
		return variable.IndexPage{}, errors.Err(err)
	}

	p := variable.IndexPage{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      string(csrf.TemplateField(r)),
		Search:    r.URL.Query().Get("search"),
		Paginator: paginator,
		Variables: vv,
	}

	return p, nil
}

func (h Variable) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	u := h.Core.User(r)

	p, err := h.indexPage(u.VariableStore(), r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	d := template.NewDashboard(&p, r.URL, h.Core.Alert(sess), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Variable) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	p := &variable.CreatePage{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Core.FormErrors(sess),
			Fields: h.Core.FormFields(sess),
		},
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(sess), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Variable) Store(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	v, err := h.Core.Store(r, sess)

	if err != nil {
		cause := errors.Cause(err)

		switch cause {
		case form.ErrValidation:
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		case model.ErrPermission:
			sess.AddFlash(template.Danger("Failed to create variable: could not add to namespace"), "alert")
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		default:
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to create variable: " + cause.Error()), "alert")
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}
	}

	sess.AddFlash(template.Success("Variable has been added: " + v.Key), "alert")
	http.Redirect(w, r, "/variables", http.StatusSeeOther)
}

func (h Variable) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	v := h.Core.Variable(r)

	if err := h.Core.Destroy(r); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		sess.AddFlash(template.Danger("Failed to delete variable: " + cause.Error()), "alert")
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	sess.AddFlash(template.Success("Variable has been deleted: " + v.Key), "alert")
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
