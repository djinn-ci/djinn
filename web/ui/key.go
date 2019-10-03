package ui

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/key"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/web/core"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
)

type Key struct {
	Core core.Key

	Keys model.KeyStore
}

func (h Key) indexPage(keys model.KeyStore, r *http.Request, opts ...query.Option) (key.IndexPage, error) {
	kk, err := h.Core.Index(keys, r, opts...)

	if err != nil {
		return key.IndexPage{}, errors.Err(err)
	}

	u := h.Core.User(r)

	search := r.URL.Query().Get("search")

	p := key.IndexPage{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:   string(csrf.TemplateField(r)),
		Search: search,
		Keys:   kk,
	}

	return p, nil
}

func (h Key) Index(w http.ResponseWriter, r *http.Request) {
	u := h.Core.User(r)

	p, err := h.indexPage(u.KeyStore(), r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	d := template.NewDashboard(&p, r.URL, h.Core.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Key) Create(w http.ResponseWriter, r *http.Request) {
	p := &key.Form{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Core.Errors(w, r),
			Fields: h.Core.Form(w, r),
		},
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Key) Store(w http.ResponseWriter, r *http.Request) {
	k, err := h.Core.Store(w, r)

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

	h.Core.FlashAlert(w, r, template.Success("Key has been added: " + k.Name))

	http.Redirect(w, r, "/keys", http.StatusSeeOther)
}

func (h Key) Edit(w http.ResponseWriter, r *http.Request) {
	k := h.Core.Key(r)

	if err := k.LoadNamespace(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &key.Form{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Core.Errors(w, r),
			Fields: h.Core.Form(w, r),
		},
		Key: k,
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Key) Update(w http.ResponseWriter, r *http.Request) {
	k, err := h.Core.Update(w, r)

	if err != nil {
		cause := errors.Cause(err)

		if cause == core.ErrValidationFailed {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		h.Core.FlashAlert(w, r, template.Danger("Failed to update  SSH key: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.Core.FlashAlert(w, r, template.Success("Key changes saved for: " + k.Name))

	http.Redirect(w, r, "/keys", http.StatusSeeOther)
}

func (h Key) Destroy(w http.ResponseWriter, r *http.Request) {
	k := h.Core.Key(r)

	if err := h.Keys.Delete(k); err != nil {
		log.Error.Println(errors.Err(err))
		h.Core.FlashAlert(w, r, template.Danger("Failed to delete SSH key: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.Core.FlashAlert(w, r, template.Success("Key has been deleted: " + k.Name))

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
