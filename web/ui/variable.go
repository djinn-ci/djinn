package ui

import (
	"database/sql"
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/variable"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"
)

type Variable struct {
	web.Handler

	Variables model.VariableStore
}

func (h Variable) Variable(r *http.Request) *model.Variable {
	val := r.Context().Value("variable")

	v, _ := val.(*model.Variable)

	return v
}

func (h Variable) Index(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	search := r.URL.Query().Get("search")

	vv, err := u.VariableStore().Index(model.Search("key", search))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &variable.IndexPage{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      string(csrf.TemplateField(r)),
		Search:    search,
		Variables: vv,
	}

	d := template.NewDashboard(p, r.URL, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Variable) Create(w http.ResponseWriter, r *http.Request) {
	p := &variable.CreatePage{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Errors(w, r),
			Fields: h.Form(w, r),
		},
	}

	d := template.NewDashboard(p, r.URL, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Variable) Store(w http.ResponseWriter, r *http.Request) {
	var err error

	u := h.User(r)

	variables := u.VariableStore()
	namespaces := u.NamespaceStore()

	f := &form.Variable{
		Variables: variables,
	}

	if err := form.Unmarshal(f, r); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create variable: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	n := &model.Namespace{}

	if f.Namespace != "" {
		n, err = namespaces.FindOrCreate(f.Namespace)

		if err != nil {
			log.Error.Println(errors.Err(err))
			h.FlashAlert(w, r, template.Danger("Failed to create variable: " + errors.Cause(err).Error()))
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		f.Variables = n.VariableStore()
	}

	if err := f.Validate(); err != nil {
		if _, ok := err.(form.Errors); ok {
			h.FlashErrors(w, r, err.(form.Errors))
			h.FlashForm(w, r, f)
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create SSH key: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	v := variables.New()
	v.NamespaceID = sql.NullInt64{
		Int64: n.ID,
		Valid: n.ID > 0,
	}
	v.Key = f.Key
	v.Value = f.Value

	if err := h.Variables.Create(v); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create variable: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Variable has been added: " + v.Key))

	http.Redirect(w, r, "/variables", http.StatusSeeOther)
}

func (h Variable) Destroy(w http.ResponseWriter, r *http.Request) {
	v := h.Variable(r)

	if err := h.Variables.Delete(v); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to delete variable: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Variable has been deleted: " + v.Key))

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
