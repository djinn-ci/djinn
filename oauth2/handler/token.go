package handler

import (
	"net/http"
	"strings"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/oauth2"
	oauth2template "github.com/andrewpillar/djinn/oauth2/template"
	"github.com/andrewpillar/djinn/template"
	"github.com/andrewpillar/djinn/user"
	usertemplate "github.com/andrewpillar/djinn/user/template"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
)

type Token struct {
	web.Handler

	Tokens *oauth2.TokenStore
}

func (h Token) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	tt, err := oauth2.NewTokenStore(h.DB, u).All(query.OrderDesc("created_at"))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
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
		Section: &oauth2template.TokenIndex{
			CSRF:   csrfField,
			Tokens: tt,
		},
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrfField))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Token) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	csrfField := string(csrf.TemplateField(r))
	f := web.FormFields(sess)

	section := &oauth2template.TokenForm{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: web.FormErrors(sess),
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
			URL: r.URL,
		},
		Section: section,
	}

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Token) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	tokens := oauth2.NewTokenStore(h.DB, u)

	f := &oauth2.TokenForm{
		Tokens: tokens,
	}

	if err := form.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create token"), "alert")
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

	sc, _ := oauth2.UnmarshalScope(strings.Join(f.Scope, " "))

	t, err := tokens.Create(f.Name, sc)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create token"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(t.ID, "token_id")
	h.Redirect(w, r, "/settings/tokens")
}

func (h Token) Edit(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	t, ok := oauth2.TokenFromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get token from request context")
	}

	f := web.FormFields(sess)

	csrfField := string(csrf.TemplateField(r))

	p := &oauth2template.TokenForm{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: web.FormErrors(sess),
			Fields: f,
		},
		Token:  t,
		Scopes: t.Permissions(),
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Token) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, _ := h.Session(r)

	u, ok := user.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	t, ok := oauth2.TokenFromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get token from request context")
	}

	if web.BasePath(r.URL.Path) == "regenerate" {
		if err := h.Tokens.Reset(t.ID); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to update token"), "alert")
			h.RedirectBack(w, r)
			return
		}

		sess.AddFlash(t.ID, "token_id")
		sess.AddFlash(template.Success("Token has been updated: "+t.Name), "alert")
		h.Redirect(w, r, "/settings/tokens")
		return
	}

	tokens := oauth2.NewTokenStore(h.DB, u)

	f := &oauth2.TokenForm{
		Tokens: tokens,
		Token:  t,
	}

	if err := form.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update token"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sc, _ := oauth2.UnmarshalScope(strings.Join(f.Scope, " "))

	if err := h.Tokens.Update(t.ID, f.Name, sc); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update token"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(t.ID, "token_id")
	sess.AddFlash(template.Success("Token has been updated: "+t.Name), "alert")
	h.Redirect(w, r, "/settings/tokens")
}

func (h Token) Destroy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, _ := h.Session(r)

	u, ok := user.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	if web.BasePath(r.URL.Path) == "revoke" {
		tokens := oauth2.NewTokenStore(h.DB, u)

		tt, err := tokens.All()

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to revoke tokens"), "alert")
			h.RedirectBack(w, r)
			return
		}

		ids := make([]int64, 0, len(tt))

		for _, t := range tt {
			ids = append(ids, t.ID)
		}

		if err := tokens.Delete(ids...); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to revoke tokens"), "alert")
			h.RedirectBack(w, r)
			return
		}
		h.RedirectBack(w, r)
		return
	}

	t, ok := oauth2.TokenFromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get token from request context")
	}

	if err := h.Tokens.Delete(t.ID); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to revoke token"), "alert")
		h.RedirectBack(w, r)
		return
	}
	h.RedirectBack(w, r)
}
