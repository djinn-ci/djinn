package handler

import (
	"crypto/rand"
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/oauth2"
	oauth2template "github.com/andrewpillar/thrall/oauth2/template"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"

	"golang.org/x/crypto/bcrypt"
)

type App struct {
	web.Handler
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

	p := &oauth2template.AppIndex{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Apps:     aa,
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

	f := &oauth2.AppForm{}

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

	hash, err := bcrypt.GenerateFromPassword(secret, bcrypt.DefaultCost)

	if err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create app"), "alert")
		return
	}

	apps := oauth2.NewAppStore(h.DB, u)

	a := apps.New()
	a.ClientID = id
	a.ClientSecret = hash
	a.Name = f.Name
	a.Description = f.Description
	a.Domain = f.Domain
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
