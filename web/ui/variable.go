package ui

import (
	"net/http"
	"strconv"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/variable"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type Variable struct {
	web.Handler
}

func (h Variable) Index(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	search := r.URL.Query().Get("search")

	vv, err := u.VariableStore().All(model.Search("key", search))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &variable.IndexPage{
		Page: template.Page{
			URI: r.URL.Path,
		},
		CSRF:      csrf.TemplateField(r),
		Search:    search,
		Variables: vv,
	}

	d := template.NewDashboard(p, r.URL.Path, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Variable) Create(w http.ResponseWriter, r *http.Request) {
	p := &variable.CreatePage{
		Form: template.Form{
			CSRF:   csrf.TemplateField(r),
			Errors: h.Errors(w, r),
			Form:   h.Form(w, r),
		},
	}

	d := template.NewDashboard(p, r.URL.Path, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Variable) Store(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	variables := u.VariableStore()

	f := &form.Variable{
		Variables: variables,
	}

	if err := h.ValidateForm(f, w, r); err != nil {
		if _, ok := err.(form.Errors); ok {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	v := variables.New()
	v.Key = f.Key
	v.Value = f.Value

	if err := v.Create(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h Variable) Destroy(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)

	id, _ := strconv.ParseInt(vars["variable"], 10, 64)

	v, err := u.VariableStore().Find(id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if v.IsZero() {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	if err := v.Destroy(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	h.FlashAlert(w, r, template.Success("Variable has been deleted: " + v.Key))

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
