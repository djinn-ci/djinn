package handler

import (
	"net/http"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/template"
	"github.com/andrewpillar/djinn/user"
	variabletemplate "github.com/andrewpillar/djinn/variable/template"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
)

// UI is the handler for handling UI requests made for variable creation, and
// management.
type UI struct {
	Variable
}

// Index serves the HTML response detailing the list of variables.
func (h UI) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	vv, paginator, err := h.IndexWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
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
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrfField)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Create serves the HTML response for creating variables via the web frontend.
func (h UI) Create(w http.ResponseWriter, r *http.Request) {
	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	sess, save := h.Session(r)

	csrfField := string(csrf.TemplateField(r))

	p := &variabletemplate.Create{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrfField)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Store validates the form submitted in the given request for creating a
// variable. If validation fails then the user is redirected back to the
// request referer, otherwise they are redirect back to the variable index.
func (h Variable) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	v, f, err := h.StoreModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		switch cause {
		case namespace.ErrName:
			errs := webutil.NewErrors()
			errs.Put("namespace", cause)

			sess.AddFlash(errs, "form_errors")
			h.RedirectBack(w, r)
			return
		case namespace.ErrPermission:
			sess.AddFlash(template.Alert{
				Level:    template.Danger,
				Close:    true,
				Message: "Failed to create variable: could not add to namespace",
			}, "alert")
			h.RedirectBack(w, r)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:    template.Danger,
				Close:    true,
				Message: "Failed to create variable",
			}, "alert")
			h.RedirectBack(w, r)
			return
		}
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Variable has been added: "+v.Key,
	}, "alert")
	h.Redirect(w, r, "/variables")
}

// Destroy removes the variable in the given request context from the database.
// This redirects back to the request's referer.
func (h Variable) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to delete variable",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}
	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Variable has been deleted",
	}, "alert")
	h.RedirectBack(w, r)
}
