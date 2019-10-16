package ui

import (
	"net/http"
	"os"
	"regexp"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/file"
	"github.com/andrewpillar/thrall/template/image"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/web/core"

	"github.com/gorilla/csrf"
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
	u := h.Core.User(r)

	p, err := h.indexPage(u.ImageStore(), r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	d := template.NewDashboard(&p, r.URL, h.Core.Alert(w, r), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Image) Create(w http.ResponseWriter, r *http.Request) {
	p := &file.CreatePage{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Core.Errors(w, r),
			Fields: h.Core.Form(w, r),
		},
		Name:   "image",
		Action: "/images",
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(w, r), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Image) Store(w http.ResponseWriter, r *http.Request) {
	i, err := h.Core.Store(w, r)

	if err != nil {
		cause := errors.Cause(err)

		if cause == core.ErrValidationFailed {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		h.Core.FlashAlert(w, r, template.Danger("Failed to create SSH key: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.Core.FlashAlert(w, r, template.Success("Image has been added: " + i.Name))

	http.Redirect(w, r, "/images", http.StatusSeeOther)
}

func (h Image) Download(w http.ResponseWriter, r *http.Request) {
	i := h.Core.Image(r)

	f, err := h.Core.FileStore.Open(i.Hash)

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
	i := h.Core.Image(r)

	if err := h.Core.Destroy(r); err != nil {
		cause := errors.Cause(err)

		if !os.IsNotExist(err) {
			log.Error.Println(errors.Err(err))
		}

		h.Core.FlashAlert(w, r, template.Danger("Failed to delete image: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.Core.FlashAlert(w, r, template.Success("Image has been deleted: " + i.Name))

	ref := r.Header.Get("Referer")

	if matched, err := regexp.Match("/images/[0-9]+", []byte(ref)); err == nil || matched {
		http.Redirect(w, r, "/images", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
