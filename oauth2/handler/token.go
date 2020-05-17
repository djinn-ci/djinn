package handler

import (
	"crypto/rand"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/oauth2"
	oauth2template "github.com/andrewpillar/thrall/oauth2/template"
	"github.com/andrewpillar/thrall/template"
	usertemplate "github.com/andrewpillar/thrall/user/template"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
)

type Token struct {
	web.Handler
}

func (h Token) Model(r *http.Request) *oauth2.Token {
	val := r.Context().Value("token")
	t, _ := val.(*oauth2.Token)
	return t
}

func (h Token) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u := h.User(r)

	tt, err := oauth2.NewTokenStore(h.DB, u).All(query.OrderDesc("created_at"))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	flashes := sess.Flashes("token_id")

	for _, t := range tt {
		if flashes != nil && len(flashes) > 0 {
			if id, _ := flashes[0].(int64); id == t.ID {
				continue
			}
		}
		t.Token = nil
	}

	csrfField := csrf.TemplateField(r)

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}

	p := &usertemplate.Settings{
		BasePage: bp,
		Section:  &oauth2template.TokenIndex{
			CSRF:     csrfField,
			Tokens:   tt,
		},
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), string(csrfField))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Token) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrfField := string(csrf.TemplateField(r))
	f := h.FormFields(sess)

	section := &oauth2template.TokenForm{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: h.FormErrors(sess),
			Fields: f,
		},
		Scopes: make(map[string]struct{}),
	}

	scope := strings.Split(f["scope"], " ")

	for _, sc := range scope {
		section.Scopes[sc] = struct{}{}
	}

	p := &usertemplate.Settings{
		BasePage: template.BasePage{
			URL:  r.URL,
		},
		Section:  section,
	}

	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Token) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u := h.User(r)

	tokens := oauth2.NewTokenStore(h.DB, u)

	f := &oauth2.TokenForm{
		Tokens: tokens,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); !ok {
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to create token"), "alert")
		}
		h.RedirectBack(w, r)
		return
	}

	if len(f.Scope) == 0 {
		for _, res := range oauth2.Resources {
			for _, perm := range oauth2.Permissions {
				f.Scope = append(f.Scope, res.String()+":"+perm.String())
			}
		}
	}

	t := tokens.New()
	t.Name = f.Name
	t.Token = make([]byte, 16)
	t.Scope, _ = oauth2.UnmarshalScope(strings.Join(f.Scope, " "))

	if _, err := rand.Read(t.Token); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create token"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := tokens.Create(t); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create token"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(t.ID, "token_id")
	h.Redirect(w, r, "/settings/tokens")
}

func (h Token) Edit(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	t := h.Model(r)
	f := h.FormFields(sess)

	csrfField := string(csrf.TemplateField(r))

	p := &oauth2template.TokenForm{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: h.FormErrors(sess),
			Fields: f,
		},
		Token:  t,
		Scopes: t.Permissions(),
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Token) Update(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u := h.User(r)
	t := h.Model(r)

	tokens := oauth2.NewTokenStore(h.DB, u)

	if filepath.Base(r.URL.Path) == "regenerate" {
		t.Token = make([]byte, 16)

		if _, err := rand.Read(t.Token); err != nil {
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to update token"), "alert")
			h.RedirectBack(w, r)
			return
		}

		if err := tokens.Update(t); err != nil {
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to update token"), "alert")
			h.RedirectBack(w, r)
			return
		}

		sess.AddFlash(t.ID, "token_id")
		sess.AddFlash(template.Success("Token has been updated: "+t.Name), "alert")
		h.Redirect(w, r, "/settings/tokens")
		return
	}

	f := &oauth2.TokenForm{
		Tokens: tokens,
		Token:  t,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); !ok {
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to update token"), "alert")
		}
		h.RedirectBack(w, r)
		return
	}

	t.Name = f.Name
	t.Scope, _ = oauth2.UnmarshalScope(strings.Join(f.Scope, " "))

	if err := tokens.Update(t); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update token"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(t.ID, "token_id")
	sess.AddFlash(template.Success("Token has been updated: "+t.Name), "alert")
	h.Redirect(w, r, "/settings/tokens")
}

func (h Token) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u := h.User(r)

	tokens := oauth2.NewTokenStore(h.DB, u)

	if filepath.Base(r.URL.Path) == "revoke" {

		tt, err := tokens.All()

		if err != nil {
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to revoke tokens"), "alert")
			h.RedirectBack(w, r)
			return
		}

		if err := tokens.Delete(tt...); err != nil {
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to revoke tokens"), "alert")
			h.RedirectBack(w, r)
			return
		}
		h.RedirectBack(w, r)
		return
	}

	if t := h.Model(r); !t.IsZero() {
		if err := tokens.Delete(t); err != nil {
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to revoke token"), "alert")
			h.RedirectBack(w, r)
			return
		}
	}
	h.RedirectBack(w, r)
}
