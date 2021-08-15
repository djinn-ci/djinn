package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"djinn-ci.com/errors"
	"djinn-ci.com/database"
	"djinn-ci.com/image"
	imagetemplate "djinn-ci.com/image/template"
	"djinn-ci.com/namespace"
	"djinn-ci.com/template"
	"djinn-ci.com/user"
	"djinn-ci.com/web"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

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

	itab := make(map[int64]*image.Image)

	for _, i := range ii {
		itab[i.ID] = i
	}

	ids := make([]interface{}, 0, len(ii))

	for _, i := range ii {
		ids = append(ids, i.ID)
	}

	dd, err := image.NewDownloadStore(h.DB, u).All(query.Where("image_id", "IN", database.List(ids...)))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	for _, d := range dd {
		i := itab[d.ID]
		i.Download = d
		itab[d.ID] = i
	}

	csrf := csrf.TemplateField(r)

	p := &imagetemplate.Index{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      csrf,
		Paginator: paginator,
		Images:    ii,
		Search:    r.URL.Query().Get("search"),
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Create serves the HTML response for creating and uploading images via the
// web frontend.
func (h Image) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	csrf := csrf.TemplateField(r)

	p := &imagetemplate.Create{
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
// image. If validation fails then the user is redirected back to the request
// referer, otherwise they are redirect back to the image index.
func (h Image) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	i, f, err := h.StoreModel(w, r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, &f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		if patherr, ok := cause.(*os.PathError); ok {
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: patherr.Error(),
			}, "alert")
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
				Message: "Failed to create image: could not add to namespace",
			}, "alert")
			h.RedirectBack(w, r)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to create image",
			}, "alert")
			h.RedirectBack(w, r)
			return
		}
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Image has been added: " + i.Name,
	}, "alert")
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

	rec, err := h.store.Open(filepath.Join(i.Driver.String(), i.Hash))

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
	w.Header().Set("Content-Type", image.MimeTypeQEMU)
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

		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to delete image",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Image has been deleted",
	}, "alert")

	if matched, _ := regexp.Match("/images/[0-9]+", []byte(r.Header.Get("Referer"))); matched {
		h.Redirect(w, r, "/images")
		return
	}
	h.RedirectBack(w, r)
}
