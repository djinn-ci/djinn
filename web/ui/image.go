package ui

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/file"
	"github.com/andrewpillar/thrall/template/image"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/web/core"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type Image struct {
	Core core.Image
}

func (h Image) indexPage(images model.ImageStore, r *http.Request) (image.IndexPage, error) {
	u := h.Core.User(r)

	ii, paginator, err := h.Core.Index(images, r)

	if err != nil {
		return image.IndexPage{}, errors.Err(err)
	}

	p := image.IndexPage{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      string(csrf.TemplateField(r)),
		Paginator: paginator,
		Images:    ii,
		Search:    r.URL.Query().Get("search"),
	}

	return p, nil
}

func (h Image) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	u := h.Core.User(r)

	p, err := h.indexPage(u.ImageStore(), r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	d := template.NewDashboard(&p, r.URL, h.Core.Alert(sess), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Image) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	p := &file.CreatePage{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Core.FormErrors(sess),
			Fields: h.Core.FormFields(sess),
		},
		Name:   "image",
		Action: "/images",
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(sess), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Image) Store(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	i, err := h.Core.Store(w, r, sess)

	if err != nil {
		cause := errors.Cause(err)

		switch cause {
		case form.ErrValidation:
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		case model.ErrPermission:
			sess.AddFlash(template.Danger("Failed to create image: could not add to namespace"), "alert")
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		default:
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to create image: " + cause.Error()))
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}
	}

	sess.AddFlash(template.Success("Image has been added: " + i.Name), "alert")

	http.Redirect(w, r, "/images", http.StatusSeeOther)
}

func (h Image) Download(w http.ResponseWriter, r *http.Request) {
	i := h.Core.Image(r)

	vars := mux.Vars(r)

	if i.Name != vars["name"] {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	f, err := h.Core.FileStore.Open(filepath.Join(i.Driver.String(), i.Name + "::" + i.Hash))

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

	http.ServeContent(w, r, i.Name, i.UpdatedAt, f)
}

func (h Image) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	i := h.Core.Image(r)

	if err := h.Core.Destroy(r); err != nil {
		cause := errors.Cause(err)

		if !os.IsNotExist(err) {
			log.Error.Println(errors.Err(err))
		}

		sess.AddFlash(template.Danger("Failed to delete image: " + cause.Error()), "alert")
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	sess.AddFlash(template.Success("Image has been deleted: " + i.Name), "alert")

	ref := r.Header.Get("Referer")

	if matched, err := regexp.Match("/images/[0-9]+", []byte(ref)); err == nil || matched {
		http.Redirect(w, r, "/images", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
