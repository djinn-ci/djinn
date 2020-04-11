package handler

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/key"
	keytemplate "github.com/andrewpillar/thrall/key/template"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"
)

type UI struct {
	Key

	Keys key.Store
}

func (h Key) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u := h.User(r)

	kk, paginator, err := h.IndexWithRelations(key.NewStore(h.DB, u), r.URL.Query())

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusNotFound)
		return
	}

	search := r.URL.Query().Get("search")
	csrfField := string(csrf.TemplateField(r))

	p := &keytemplate.Index{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      csrfField,
		Search:    search,
		Paginator: paginator,
		Keys:      kk,
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Key) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)
	csrfField := string(csrf.TemplateField(r))

	p := &keytemplate.Form{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: h.FormErrors(sess),
			Fields: h.FormFields(sess),
		},
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Key) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	k, err := h.StoreModel(r, sess)

	if err != nil {
		if _, ok := err.(form.Errors); ok {
			h.RedirectBack(w, r)
			return
		}

		if errors.Cause(err) == namespace.ErrPermission {
			sess.AddFlash(template.Danger("Failed to create key: could not add to namespace"), "alert")
		} else {
			sess.AddFlash(template.Danger("Failed to create key"), "alert")
			log.Error.Println(errors.Err(err))
		}
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Key has been added: "+k.Name), "alert")
	h.Redirect(w, r, "/keys")
}

func (h Key) Edit(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	k, err := h.ShowWithRelations(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	csrfField := string(csrf.TemplateField(r))

	p := &keytemplate.Form{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: h.FormErrors(sess),
			Fields: h.FormFields(sess),
		},
		Key: k,
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Key) Update(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	k, err := h.UpdateModel(r, sess)

	if err != nil {
		if _, ok := err.(form.Errors); ok {
			h.RedirectBack(w, r)
			return
		}

		if errors.Cause(err) == namespace.ErrPermission {
			sess.AddFlash(template.Danger("Failed to update key: could not add to namespace"), "alert")
			h.RedirectBack(w, r)
			return
		}
		log.Error.Println(errors.Err(err))
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Key has been updated: "+k.Name), "alert")
	h.Redirect(w, r, "/keys")
}

func (h Key) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)
	k := h.Model(r)

	if err := h.Keys.Delete(k); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to delete key"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Key has been deleted: "+k.Name), "alert")
	h.RedirectBack(w, r)
}
