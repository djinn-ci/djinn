package handler

import (
	"net/http"
	"strings"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/oauth2"
	oauth2template "github.com/andrewpillar/djinn/oauth2/template"
	"github.com/andrewpillar/djinn/template"
	"github.com/andrewpillar/djinn/user"
	usertemplate "github.com/andrewpillar/djinn/user/template"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
)

// Token handles serving responses for managing personal access tokens that a
// user creates to access the API.
type Token struct {
	web.Handler

	tokens *oauth2.TokenStore
}

func NewToken(h web.Handler) Token {
	return Token{
		tokens: oauth2.NewTokenStore(h.DB),
	}
}

// Index serves the HTML response detailing the list of personal access tokens
// created for the current user.
func (h Token) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	tt, err := oauth2.NewTokenStore(h.DB, u).All(
		query.Where("app_id", "IS", query.Lit("NULL")),
		query.OrderDesc("created_at"),
	)

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
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Create serves the HTML response for creating a new personal access token.
func (h Token) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	csrfField := string(csrf.TemplateField(r))
	f := webutil.FormFields(sess)

	section := &oauth2template.TokenForm{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: webutil.FormErrors(sess),
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
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Store validates the form submitted in the given request for creating a
// personal access token. If validation fails then the user is redirected back
// to the request referer, otherwise they are redirect back to the token index.
// If no specific scopes are specified in the form submission, then all scopes
// are added to the created token.
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

	if err := webutil.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
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

// Edit serves the HTML response for editing a personal access token.
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

	f := webutil.FormFields(sess)

	csrfField := string(csrf.TemplateField(r))

	p := &oauth2template.TokenForm{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: webutil.FormErrors(sess),
			Fields: f,
		},
		Token:  t,
		Scopes: t.Permissions(),
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrfField)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Update validates the form submitted in the given request for updating token.
// If validation fails then the user is redirect back to the request's referer,
// otherwise they are redirected back to the token index. If the base path of
// the request URL is "/regenerate", then the token is simply regenerated.
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

	if webutil.BasePath(r.URL.Path) == "regenerate" {
		if err := h.tokens.Reset(t.ID); err != nil {
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

	if err := webutil.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update token"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sc, _ := oauth2.UnmarshalScope(strings.Join(f.Scope, " "))

	if err := h.tokens.Update(t.ID, f.Name, sc); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update token"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(t.ID, "token_id")
	sess.AddFlash(template.Success("Token has been updated: "+t.Name), "alert")
	h.Redirect(w, r, "/settings/tokens")
}

// Destroy removes the token in the request from the database.
func (h Token) Destroy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, _ := h.Session(r)

	u, ok := user.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	if webutil.BasePath(r.URL.Path) == "revoke" {
		tokens := oauth2.NewTokenStore(h.DB, u)

		tt, err := tokens.All(query.Where("app_id", "IS", query.Lit("NULL")))

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

	if err := h.tokens.Delete(t.ID); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to revoke token"), "alert")
		h.RedirectBack(w, r)
		return
	}
	h.RedirectBack(w, r)
}
