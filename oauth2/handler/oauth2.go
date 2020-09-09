package handler

import (
	"bytes"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/oauth2"
	oauth2template "github.com/andrewpillar/djinn/oauth2/template"
	"github.com/andrewpillar/djinn/template"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
)

type Oauth2 struct {
	web.Handler

	Apps   *oauth2.AppStore
	Tokens *oauth2.TokenStore
}

func (h Oauth2) handleAuthPage(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	q := r.URL.Query()

	clientId := q.Get("client_id")
	redirectUri := q.Get("redirect_uri")
	state := q.Get("state")

	if clientId == "" {
		web.HTMLError(w, "No client ID in request", http.StatusBadRequest)
		return
	}

	scope, err := oauth2.UnmarshalScope(q.Get("scope"))

	if err != nil {
		web.HTMLError(w, errors.Cause(err).Error(), http.StatusBadRequest)
		return
	}

	b, err := hex.DecodeString(clientId)

	if err != nil {
		web.HTMLError(w, errors.Cause(err).Error(), http.StatusBadRequest)
		return
	}

	a, err := h.Apps.Get(query.Where("client_id", "=", b))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	// User is logged in, so check for an existing token and update the scope
	// for the token if a new scope is requested.
	if !u.IsZero() {
		t, err := oauth2.NewTokenStore(h.DB, u, a).Get()

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if !t.IsZero() {
			diff := oauth2.ScopeDiff(scope, t.Scope)

			// New scope has been requested, so goto auth login response.
			if len(diff) > 0 {
				goto resp
			}

			scope = append(scope, diff...)

			if len(scope) == 0 {
				scope = t.Scope
			}

			c, err := oauth2.NewCodeStore(h.DB, u).Create(scope)

			if err != nil {
				h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
				web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
				return
			}

			redirectQuery := url.Values(make(map[string][]string))
			redirectQuery.Add("code", hex.EncodeToString(c.Code))

			if state != "" {
				redirectQuery.Add("state", state)
			}
			http.Redirect(w, r, redirectUri+"?"+redirectQuery.Encode(), http.StatusSeeOther)
			return
		}
	}

	if len(scope) == 0 {
		web.HTMLError(w, "No scope requested", http.StatusBadRequest)
		return
	}

resp:
	p := &oauth2template.Auth{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: web.FormErrors(sess),
			Fields: web.FormFields(sess),
		},
		User:        u,
		Name:        a.Name,
		ClientID:    clientId,
		RedirectURI: a.RedirectURI,
		State:       state,
		Scope:       scope,
	}
	save(r, w)
	web.HTML(w, template.Render(p), http.StatusOK)
}

func (h Oauth2) Auth(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		h.handleAuthPage(w, r)
		return
	}

	sess, _ := h.Session(r)

	f := &oauth2.AuthorizeForm{}

	if err := form.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		h.RedirectBack(w, r)
		return
	}

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	var err error

	clientId, err := hex.DecodeString(f.ClientID)

	if err != nil {
		errs := form.NewErrors()
		errs.Put("client_id", err)

		sess.AddFlash(errs, "form_errorS")
		sess.AddFlash(f.Fields(), "form_fields")
		h.RedirectBack(w, r)
		return
	}

	a, err := h.Apps.Get(query.Where("client_id", "=", clientId))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if f.Authenticate {
		u, err = h.Users.Auth(f.Handle, f.Password)

		if err != nil {
			if errors.Cause(err) != user.ErrAuth {
				h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
				web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
				return
			}

			errs := form.NewErrors()
			errs.Put("handle", user.ErrAuth)
			errs.Put("password", user.ErrAuth)

			sess.AddFlash(errs, "form_errors")
			sess.AddFlash(f.Fields(), "form_fields")
			h.RedirectBack(w, r)
			return
		}
	}

	if u.IsZero() {
		web.HTMLError(w, "No user in request", http.StatusBadRequest)
		return
	}

	scope, err := oauth2.UnmarshalScope(f.Scope)

	if err != nil {
		web.HTMLError(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	c, err := oauth2.NewCodeStore(h.DB, u, a).Create(scope)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	redirectQuery := url.Values(make(map[string][]string))
	redirectQuery.Add("code", hex.EncodeToString(c.Code))

	if f.State != "" {
		redirectQuery.Add("state", f.State)
	}
	http.Redirect(w, r, f.RedirectURI+"?"+redirectQuery.Encode(), http.StatusSeeOther)
}

func (h Oauth2) Token(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}
	io.Copy(buf, r.Body)

	q, err := url.ParseQuery(buf.String())

	if err != nil {
		web.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	a, err := h.Apps.Auth(q.Get("client_id"), q.Get("client_secret"))

	if err != nil {
		if errors.Cause(err) == oauth2.ErrAuth {
			web.JSONError(w, "invalid client id and secret", http.StatusBadRequest)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, errors.Cause(err).Error(), http.StatusInternalServerError)
		return
	}

	code, err := hex.DecodeString(q.Get("code"))

	if err != nil {
		web.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	codes := oauth2.NewCodeStore(h.DB, a)
	c, err := codes.Get(query.Where("code", "=", code))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, errors.Cause(err).Error(), http.StatusInternalServerError)
		return
	}

	if c.IsZero() {
		web.JSONError(w, "Not found", http.StatusNotFound)
		return
	}

	if c.ExpiresAt.Sub(time.Now()) > time.Minute*10 {
		web.JSONError(w, "code expired", http.StatusBadRequest)
		return
	}

	u, err := h.Users.Get(query.Where("id", "=", c.UserID))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, errors.Cause(err).Error(), http.StatusInternalServerError)
		return
	}

	tokens := oauth2.NewTokenStore(h.DB, a, u)

	t, err := tokens.Get()

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, errors.Cause(err).Error(), http.StatusInternalServerError)
		return
	}

	if t.IsZero() {
		_, err := tokens.Create("client."+strconv.FormatInt(u.ID, 10), c.Scope)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	} else {
		if err := tokens.Update(t.ID, t.Name, c.Scope); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	if err := codes.Delete(c.ID); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body := map[string]string{
		"access_token": hex.EncodeToString(t.Token),
		"token_type":   "bearer",
		"scope":        t.Scope.String(),
	}

	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		web.JSON(w, body, http.StatusOK)
		return
	}

	vals := make(url.Values)

	for k, v := range body {
		vals[k] = append([]string(nil), v)
	}
	web.Text(w, vals.Encode(), http.StatusOK)
}

func (h Oauth2) Revoke(w http.ResponseWriter, r *http.Request) {
	prefix := "Bearer "
	tok := r.Header.Get("Authorization")

	b, err := hex.DecodeString(tok[len(prefix):])

	if err != nil {
		web.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	t, err := h.Tokens.Get(query.Where("token", "=", b))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if t.IsZero() {
		goto resp
	}

	if err := h.Tokens.Delete(t.ID); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
resp:
	w.WriteHeader(http.StatusNoContent)
}
