package handler

import (
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/andrewpillar/thrall/build"
	buildtemplate "github.com/andrewpillar/thrall/build/template"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/object"
	objecttemplate "github.com/andrewpillar/thrall/object/template"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type UI struct {
	Object
}

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

	csrfField := string(csrf.TemplateField(r))

	p := &objecttemplate.Index{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      csrfField,
		Paginator: paginator,
		Objects:   oo,
		Search:    r.URL.Query().Get("search"),
	}
	d := template.NewDashboard(p, r.URL, web.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrfField := string(csrf.TemplateField(r))

	p := &objecttemplate.Create{
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

func (h UI) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	o, f, err := h.StoreModel(w, r)

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
			sess.AddFlash(template.Danger("Failed to create object: could not add to namespace"), "alert")
			h.RedirectBack(w, r)
			return
		default:
			h.Log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to create object"), "alert")
			h.RedirectBack(w, r)
			return
		}
	}

	sess.AddFlash(template.Success("Object has been added: "+o.Name), "alert")
	h.Redirect(w, r, "/objects")
}

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

		rec, err := h.BlockStore.Open(o.Hash)

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

	bb, paginator, err := h.Builds.Index(
		r.URL.Query(),
		query.WhereQuery(
			"id", "IN", build.SelectObject("build_id", query.Where("object_id", "=", o.ID)),
		),
	)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := build.LoadRelations(h.Loaders, bb...); err != nil {
		h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()

	csrfField := string(csrf.TemplateField(r))

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}
	p := &objecttemplate.Show{
		BasePage: bp,
		CSRF:     csrfField,
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
	d := template.NewDashboard(p, r.URL, web.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.DeleteModel(r); err != nil {
		if os.IsNotExist(errors.Cause(err)) {
			goto resp
		}
		h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to delete object"), "alert")
		h.RedirectBack(w, r)
		return
	}

resp:
	sess.AddFlash(template.Success("Object has been deleted"), "alert")

	if matched, _ := regexp.Match("/objects/[0-9]+", []byte(r.Header.Get("Referer"))); matched {
		h.Redirect(w, r, "/objects")
		return
	}
	h.RedirectBack(w, r)
}
