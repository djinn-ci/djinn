package ui

import (
	"crypto/rand"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/model/types"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/token"

	"github.com/gorilla/csrf"
)

type Token struct {
	web.Handler

	Tokens model.TokenStore
}

func (h Token) token(r *http.Request) *model.Token {
	val := r.Context().Value("token")
	t, _ := val.(*model.Token)
	return t
}

func (h Token) Index(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	tt, err := u.TokenStore().All()

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	for _, t := range tt {
		val := h.FlashGet(w, r, "token_id")

		if val != nil {
			if id, _ := val.(int64); id == t.ID {
				continue
			}
		}

		t.Token = nil
	}

	p := &token.IndexPage{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:   string(csrf.TemplateField(r)),
		Tokens: tt,
	}

	d := template.NewDashboard(p, r.URL, h.Alert(w, r), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Token) Create(w http.ResponseWriter, r *http.Request) {
	f := h.Form(w, r)

	p := &token.Form{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Errors(w, r),
			Fields: f,
		},
		Scopes: make(map[string]struct{}),
	}

	scope := strings.Split(f["scope"], " ")

	for _, sc := range scope {
		p.Scopes[sc] = struct{}{}
	}

	d := template.NewDashboard(p, r.URL, h.Alert(w, r), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Token) Store(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	tokens := u.TokenStore()

	f := &form.Token{
		Tokens: tokens,
	}

	if err := h.ValidateForm(f, w, r); err != nil {
		if _, ok := err.(form.Errors); !ok {
			log.Error.Println(errors.Err(err))
		}
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if len(f.Scope) == 0 {
		for _, res := range types.Resources {
			for _, perm := range types.Permissions {
				f.Scope = append(f.Scope, res.String()+":"+perm.String())
			}
		}
	}

	t := tokens.New()
	t.Name = f.Name
	t.Token = make([]byte, 16)
	t.Scope, _ = types.UnmarshalScope(strings.Join(f.Scope, " "))

	if _, err := rand.Read(t.Token); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := tokens.Create(t); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	h.FlashPut(w, r, "token_id", t.ID)

	http.Redirect(w, r, "/settings/tokens", http.StatusSeeOther)
}

func (h Token) Edit(w http.ResponseWriter, r *http.Request) {
	t := h.token(r)
	f := h.Form(w, r)

	p := &token.Form{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Errors(w, r),
			Fields: f,
		},
		Token:  t,
		Scopes: t.Permissions(),
	}

	d := template.NewDashboard(p, r.URL, h.Alert(w, r), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Token) Update(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)
	t := h.token(r)

	f := &form.Token{
		Tokens: u.TokenStore(),
		Token:  t,
	}

	if err := h.ValidateForm(f, w, r); err != nil {
		if _, ok := err.(form.Errors); !ok {
			log.Error.Println(errors.Err(err))
		}
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	t.Name = f.Name
	t.Scope, _ = types.UnmarshalScope(strings.Join(f.Scope, " "))

	if filepath.Base(r.URL.Path) == "regenerate" {
		t.Token = make([]byte, 16)

		rand.Read(t.Token)
	}

	if err := h.Tokens.Update(t); err != nil {
		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to update token: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashPut(w, r, "token_id", t.ID)

	h.FlashAlert(w, r, template.Success("Token has been updated: " + t.Name))
	http.Redirect(w, r, "/settings/tokens", http.StatusSeeOther)
}

func (h Token) Destroy(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)
	tokens := u.TokenStore()

	if filepath.Base(r.URL.Path) == "revoke" {
		tt, err := tokens.All()

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if err := tokens.Delete(tt...); err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if t := h.token(r); !t.IsZero() {
		if err := tokens.Delete(t); err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	return
}
