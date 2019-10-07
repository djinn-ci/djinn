package ui

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"
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
	u := h.Core.User(r)

	p, err := h.indexPage(u.VariableStore(), r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	d := template.NewDashboard(&p, r.URL, h.Core.Alert(w, r), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Variable) Create(w http.ResponseWriter, r *http.Request) {
	p := &variable.CreatePage{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Core.Errors(w, r),
			Fields: h.Core.Form(w, r),
		},
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(w, r), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Variable) Store(w http.ResponseWriter, r *http.Request) {
	v, err := h.Core.Store(w, r)

	if err != nil {
		cause := errors.Cause(err)

		if cause == core.ErrValidationFailed {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		h.Core.FlashAlert(w, r, template.Danger("Failed to create variable: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.Core.FlashAlert(w, r, template.Success("Variable has been added: " + v.Key))

	http.Redirect(w, r, "/variables", http.StatusSeeOther)
}

func (h Variable) Destroy(w http.ResponseWriter, r *http.Request) {
	v := h.Core.Variable(r)

	if err := h.Core.Destroy(r); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.Core.FlashAlert(w, r, template.Danger("Failed to delete variable: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.Core.FlashAlert(w, r, template.Success("Variable has been deleted: " + v.Key))

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
