package handler

import (
	"crypto/rand"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/oauth2"
	oauth2template "github.com/andrewpillar/thrall/oauth2/template"
	"github.com/andrewpillar/thrall/template"
	usertemplate "github.com/andrewpillar/thrall/user/template"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type App struct {
	web.Handler
}

func (h App) app(r *http.Request) (*oauth2.App, error) {
	u := h.User(r)

	id, err := strconv.ParseInt(mux.Vars(r)["app"], 10, 64)

	if err != nil {
		return &oauth2.App{}, model.ErrNotFound
	}

	a, err := oauth2.NewAppStore(h.DB, u).Get(query.Where("id", "=", id))

	if err != nil {
		return a, errors.Err(err)
	}

	if a.IsZero() {
		return a, model.ErrNotFound
	}
	return a, nil
}

func (h App) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u := h.User(r)

	aa, err := oauth2.NewAppStore(h.DB, u).All()

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	csrfField := string(csrf.TemplateField(r))

	p := &usertemplate.Settings{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Section: &oauth2template.AppIndex{
			BasePage: template.BasePage{
				URL:  r.URL,
				User: u,
			},
			Apps:     aa,
		},
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h App) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrfField := string(csrf.TemplateField(r))

	p := &oauth2template.AppForm{
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

func (h App) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u := h.User(r)

	f := &oauth2.AppForm{
		Apps: oauth2.NewAppStore(h.DB, u),
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); !ok {
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to create app"), "alert")
		}
		h.RedirectBack(w, r)
		return
	}

	id := make([]byte, 16)
	secret := make([]byte, 32)

	if _, err := rand.Read(id); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create app"), "alert")
		return
	}

	if _, err := rand.Read(secret); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create app"), "alert")
		return
	}

	enc, _ := crypto.Encrypt(secret)

	apps := oauth2.NewAppStore(h.DB, u)

	a := apps.New()
	a.ClientID = id
	a.ClientSecret = enc
	a.Name = f.Name
	a.Description = f.Description
	a.HomeURI = f.HomepageURI
	a.RedirectURI = f.RedirectURI

	if err := apps.Create(a); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create app"), "alert")
		return
	}

	sess.AddFlash(template.Success("Created OAuth App "+a.Name), "alert")
	h.Redirect(w, r, "/settings/apps")
}

func (h App) Show(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	a, err := h.app(r)

	if err != nil {
		if err == model.ErrNotFound {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	dec, _ := crypto.Decrypt(a.ClientSecret)

	a.ClientSecret = dec

	csrfField := string(csrf.TemplateField(r))

	p := &oauth2template.AppForm{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: h.FormErrors(sess),
			Fields: h.FormFields(sess),
		},
		App:    a,
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h App) Update(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	a, err := h.app(r)

	if err != nil {
		if err == model.ErrNotFound {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update app"), "alert")
		h.RedirectBack(w, r)
		return
	}

	base := filepath.Base(r.URL.Path)

	if base == "revoke" {
		tokens := oauth2.NewTokenStore(h.DB, a)

		tt, err := tokens.All()

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to update app"), "alert")
			h.RedirectBack(w, r)
			return
		}

		if err := tokens.Delete(tt...); err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to revoke tokens"), "alert")
			h.RedirectBack(w, r)
			return
		}
		sess.AddFlash(template.Success("App tokens revoked"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if base == "reset" {
		secret := make([]byte, 32)

		if _, err := rand.Read(secret); err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to reset secret"), "alert")
			h.RedirectBack(w, r)
			return
		}

		enc, _ := crypto.Encrypt(secret)

		a.ClientSecret = enc

		if err := oauth2.NewAppStore(h.DB).Update(a); err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to reset secret"), "alert")
			h.RedirectBack(w, r)
			return
		}
		sess.AddFlash(template.Success("App client secret reset"), "alert")
		h.RedirectBack(w, r)
		return
	}

	u := h.User(r)

	f := &oauth2.AppForm{
		Apps: oauth2.NewAppStore(h.DB, u),
		App:  a,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); !ok {
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to update app"), "alert")
		}
		h.RedirectBack(w, r)
		return
	}

	a.Name = f.Name
	a.Description = f.Description
	a.HomeURI = f.HomepageURI
	a.RedirectURI = f.RedirectURI

	if err := oauth2.NewAppStore(h.DB).Update(a); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update app"), "alert")
		return
	}

	sess.AddFlash(template.Success("App changes have been saved"), "alert")
	h.Redirect(w, r, a.Endpoint())
}
