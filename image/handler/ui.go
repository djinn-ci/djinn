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
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type UI struct {
	Image
}

func (h Image) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u := h.User(r)

	ii, paginator, err := h.IndexWithRelations(image.NewStore(h.DB, u), r)

	if err != nil {
		log.Error.Println(errors.Err(err))
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
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Image) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrfField := string(csrf.TemplateField(r))

	f := template.Form{
		CSRF:   csrfField,
		Errors: h.FormErrors(sess),
		Fields: h.FormFields(sess),
	}

	p := &imagetemplate.Create{
		Form:     f,
		FileForm: &template.FileForm{
			Form: f,
		},
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Image) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	i, err := h.StoreModel(w, r, sess)

	if err != nil {
		cause := errors.Cause(err)

		if _, ok := cause.(form.Errors); ok {
			h.RedirectBack(w, r)
			return
		}

		if cause == namespace.ErrPermission {
			sess.AddFlash(template.Danger("Failed to create image: could not add to namespace"), "alert")
			h.RedirectBack(w, r)
			return
		}

		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create image"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Image has been added: "+i.Name), "alert")
	h.Redirect(w, r, "/images")
}

func (h Image) Show(w http.ResponseWriter, r *http.Request) {
	i := h.Model(r)
	vars := mux.Vars(r)

	if i.Name != vars["name"] {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	f, err := h.FileStore.Open(filepath.Join(i.Driver.String(), i.Name+"::"+i.Hash))

	if err != nil {
		if os.IsNotExist(errors.Cause(err)) {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	defer f.Close()

	http.ServeContent(w, r, i.Name, i.CreatedAt, f)
}

func (h Image) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)
	i := h.Model(r)

	if err := h.Delete(r); err != nil {
		if !os.IsNotExist(errors.Cause(err)) {
			log.Error.Println(errors.Err(err))
		}
		sess.AddFlash(template.Danger("Failed to delete image"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Image has been deleted: "+i.Name), "alert")

	if matched, _ := regexp.Match("/images/[0-9]+", []byte(r.Header.Get("Referer"))); matched {
		h.Redirect(w, r, "/images")
		return
	}
	h.RedirectBack(w, r)
}
