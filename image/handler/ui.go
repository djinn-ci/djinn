package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/image"
	imagetemplate "github.com/andrewpillar/djinn/image/template"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/template"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

// UI is the handler for handling UI requests made for image creation, and
// management.
type UI struct {
	Image
}

// Index serves the HTML response detailing the list of images.
func (h Image) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	ii, paginator, err := h.IndexWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	csrfField := string(csrf.TemplateField(r))

	p := &imagetemplate.Index{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      csrfField,
		Paginator: paginator,
		Images:    ii,
		Search:    r.URL.Query().Get("search"),
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

// Create serves the HTML response for creating and uploading images via the
// web frontend.
func (h Image) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	csrfField := string(csrf.TemplateField(r))

	p := &imagetemplate.Create{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: web.FormErrors(sess),
			Fields: web.FormFields(sess),
		},
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

// Store validates the form submitted in the given request for creating an
// image. If validation fails then the user is redirected back to the request
// referer, otherwise they are redirect back to the image index.
func (h Image) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	i, f, err := h.StoreModel(w, r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, &f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		switch cause {
		case namespace.ErrName:
			errs := form.NewErrors()
			errs.Put("namespace", cause)

			sess.AddFlash(errs, "form_errors")
			h.RedirectBack(w, r)
			return
		case namespace.ErrPermission:
			sess.AddFlash(template.Danger("Failed to create image: could not add to namespace"), "alert")
			h.RedirectBack(w, r)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to create image"), "alert")
			h.RedirectBack(w, r)
			return
		}
	}

	sess.AddFlash(template.Success("Image has been added: "+i.Name), "alert")
	h.Redirect(w, r, "/images")
}

// Show serves the HTML response for viewing an individual image in the given
// request.
func (h Image) Show(w http.ResponseWriter, r *http.Request) {
	i, ok := image.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get image from request context")
	}

	vars := mux.Vars(r)

	if i.Name != vars["name"] {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	rec, err := h.BlockStore.Open(filepath.Join(i.Driver.String(), i.Hash))

	if err != nil {
		if os.IsNotExist(errors.Cause(err)) {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	defer rec.Close()
	http.ServeContent(w, r, i.Name, i.CreatedAt, rec)
}

// Destroy removes the image in the given request context from the database.
// This redirects back to the image index if this was done from an individual
// image view, otherwise it redirects back to the request's referer.
func (h Image) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.DeleteModel(r); err != nil {
		if !os.IsNotExist(errors.Cause(err)) {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		}
		sess.AddFlash(template.Danger("Failed to delete image"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Image has been deleted"), "alert")

	if matched, _ := regexp.Match("/images/[0-9]+", []byte(r.Header.Get("Referer"))); matched {
		h.Redirect(w, r, "/images")
		return
	}
	h.RedirectBack(w, r)
}
