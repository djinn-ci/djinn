package ui

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
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
	kk, paginator, err := h.Core.Index(keys, r, opts...)

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
		CSRF:      string(csrf.TemplateField(r)),
		Search:    search,
		Paginator: paginator,
		Keys:      kk,
	}

	return p, nil
}

func (h Key) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	u := h.Core.User(r)

	p, err := h.indexPage(u.KeyStore(), r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	d := template.NewDashboard(&p, r.URL, h.Core.Alert(sess), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Key) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	p := &key.Form{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Core.FormErrors(sess),
			Fields: h.Core.FormFields(sess),
		},
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(sess), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Key) Store(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	k, err := h.Core.Store(r, sess)

	if err != nil {
		cause := errors.Cause(err)

		switch cause {
		case form.ErrValidation:
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		case model.ErrPermission:
			sess.AddFlash(template.Danger("Failed to create key: could not add to namespace"), "alert")
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		default:
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to create key: " + cause.Error()), "alert")
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}
	}

	sess.AddFlash(template.Success("Key has been added: " + k.Name), "alert")
	http.Redirect(w, r, "/keys", http.StatusSeeOther)
}

func (h Key) Edit(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	k := h.Core.Key(r)

	if err := k.LoadNamespace(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &key.Form{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Core.FormErrors(sess),
			Fields: h.Core.FormFields(sess),
		},
		Key: k,
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(sess), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Key) Update(w http.ResponseWriter, r *http.Request) {
	sess, save:= h.Core.Session(r)
	defer save(r, w)

	k, err := h.Core.Update(r, sess)

	if err != nil {
		cause := errors.Cause(err)

		switch cause {
		case form.ErrValidation:
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		case model.ErrPermission:
			sess.AddFlash(template.Danger("Failed to update key: could not add to namespace"), "alert")
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		default:
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to update key: " + cause.Error()), "alert")
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}
	}

	sess.AddFlash(template.Success("Key changes saved for: " + k.Name), "alert")
	http.Redirect(w, r, "/keys", http.StatusSeeOther)
}

func (h Key) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	k := h.Core.Key(r)

	if err := h.Keys.Delete(k); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to delete SSH key: " + errors.Cause(err).Error()), "alert")
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	sess.AddFlash(template.Success("Key has been deleted: " + k.Name), "alert")
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
