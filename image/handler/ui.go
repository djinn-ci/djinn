package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/image"
	imagetemplate "github.com/andrewpillar/thrall/image/template"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type UI struct {
	Image
}

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
	d := template.NewDashboard(p, r.URL, web.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Image) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrfField := string(csrf.TemplateField(r))

	p := &imagetemplate.Create{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: web.FormErrors(sess),
			Fields: web.FormFields(sess),
		},
	}
	d := template.NewDashboard(p, r.URL, web.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

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
