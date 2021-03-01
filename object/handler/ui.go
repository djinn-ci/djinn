package handler

import (
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/andrewpillar/djinn/build"
	buildtemplate "github.com/andrewpillar/djinn/build/template"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/object"
	objecttemplate "github.com/andrewpillar/djinn/object/template"
	"github.com/andrewpillar/djinn/template"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

// UI is the handler for handling UI requests made for object creation, and
// management.
type UI struct {
	Object
}

// Index serves the HTML response detailing the list of objects.
func (h UI) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	oo, paginator, err := h.IndexWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	csrf := csrf.TemplateField(r)

	p := &objecttemplate.Index{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      csrf,
		Paginator: paginator,
		Objects:   oo,
		Search:    r.URL.Query().Get("search"),
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Create serves the HTML response for creating and uploading objects via the
// web frontend.
func (h UI) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	csrf := csrf.TemplateField(r)

	p := &objecttemplate.Create{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Store validates the form submitted in the given request for creating an
// object. If validation fails then the user is redirected back to the request
// referer, otherwise they are redirect back to the object index.
func (h UI) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	o, f, err := h.StoreModel(w, r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, &f, ferrs)
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
				Message: "Failed to create object: could not add to namespace",
			}, "alert")
			h.RedirectBack(w, r)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to create object",
			}, "alert")
			h.RedirectBack(w, r)
			return
		}
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Object has been added: " + o.Name,
	}, "alert")
	h.Redirect(w, r, "/objects")
}

// Show serves the HTML response for viewing an individual object in the given
// request. If the penultimate part of the request URL path is "download" then
// the object content is sent in the response body.
func (h UI) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	parts := strings.Split(r.URL.Path, "/")

	if parts[len(parts)-2] == "download" {
		o, ok := object.FromContext(ctx)

		if !ok {
			h.Log.Error.Println(r.Method, r.URL, "failed to get object from request context")
		}

		if o.Name != mux.Vars(r)["name"] {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		store, err := h.store.Partition(o.UserID)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		rec, err := store.Open(o.Hash)

		if err != nil {
			if os.IsNotExist(errors.Cause(err)) {
				web.HTMLError(w, "Not found", http.StatusNotFound)
				return
			}
			h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		defer rec.Close()

		http.ServeContent(w, r, o.Name, o.CreatedAt, rec)
		return
	}

	sess, save := h.Session(r)

	u, ok := user.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	o, err := h.ShowWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	bb, paginator, err := build.NewStore(h.DB).Index(
		r.URL.Query(),
		query.Where(
			"id", "IN", build.SelectObject("build_id", query.Where("object_id", "=", query.Arg(o.ID))),
		),
	)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := build.LoadRelations(h.loaders, bb...); err != nil {
		h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()

	csrf := csrf.TemplateField(r)

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}
	p := &objecttemplate.Show{
		BasePage: bp,
		CSRF:     csrf,
		Object:   o,
		Section: &buildtemplate.Index{
			BasePage:  bp,
			Paginator: paginator,
			Builds:    bb,
			Search:    q.Get("search"),
			Status:    q.Get("status"),
			Tag:       q.Get("tag"),
		},
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Destroy removes the object in the given request context from the database.
// This redirects back to the object index if this was done from an individual
// object view, otherwise it redirects back to the request's referer.
func (h UI) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.DeleteModel(r); err != nil {
		if !os.IsNotExist(errors.Cause(err)) {
			h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to delete object",
			}, "alert")
			h.RedirectBack(w, r)
			return
		}
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Object has been deleted",
	}, "alert")

	if matched, _ := regexp.Match("/objects/[0-9]+", []byte(r.Header.Get("Referer"))); matched {
		h.Redirect(w, r, "/objects")
		return
	}
	h.RedirectBack(w, r)
}
