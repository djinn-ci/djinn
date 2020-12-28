package handler

import (
	"net/http"

	"github.com/andrewpillar/djinn/errors"
	keytemplate "github.com/andrewpillar/djinn/key/template"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/template"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
)

// UI is the handler for handling UI requests made for key creation, and
// management.
type UI struct {
	Key
}

// Index serves the HTML response detailing the list of SSH keys.
func (h Key) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	kk, paginator, err := h.IndexWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusNotFound)
		return
	}

	search := r.URL.Query().Get("search")
	csrfField := string(csrf.TemplateField(r))

	p := &keytemplate.Index{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      csrfField,
		Search:    search,
		Paginator: paginator,
		Keys:      kk,
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrfField)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Create serves the HTML response for creating SSH keys via the web frontend.
func (h Key) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	csrfField := string(csrf.TemplateField(r))

	p := &keytemplate.Form{
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

// Store validates the form submitted in the given request for creating a key.
// If validation fails then the user is redirected back to the request referer,
// otherwise they are redirect back to the key index.
func (h Key) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	k, f, err := h.StoreModel(r)

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
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to create key: could not add to namespace",
			}, "alert")
			h.RedirectBack(w, r)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to create key",
			}, "alert")
			h.RedirectBack(w, r)
			return
		}
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Key has been added: " + k.Name,
	}, "alert")
	h.Redirect(w, r, "/keys")
}

// Edit serves the HTML response for editing the key in the given request
// context.
func (h Key) Edit(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	k, err := h.ShowWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	csrfField := string(csrf.TemplateField(r))

	p := &keytemplate.Form{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
		Key: k,
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrfField)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Update validates the form submitted in the given request for updating a key.
// If validation fails then the user is redirect back to the request's referer,
// otherwise they are redirected back to the updated cron job.
func (h Key) Update(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	k, f, err := h.UpdateModel(r)

	if err != nil {
		if ferrs, ok := err.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		if errors.Cause(err) == namespace.ErrPermission {
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to update key: could not add to namespace",
			}, "alert")
			h.RedirectBack(w, r)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Key has been updated: "+k.Name,
	}, "alert")
	h.Redirect(w, r, "/keys")
}

// Destroy removes the key in the given request context from the database.
// This redirects back to the request's referer.
func (h Key) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to delete key",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}
	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Key has been deleted",
	}, "alert")
	h.RedirectBack(w, r)
}
