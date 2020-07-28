package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/oauth2"
	oauth2template "github.com/andrewpillar/thrall/oauth2/template"
	"github.com/andrewpillar/thrall/provider"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type Oauth2 struct {
	web.Handler

	Apps      *oauth2.AppStore
	Tokens    *oauth2.TokenStore
	Providers map[string]oauth2.Provider
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

// AuthClient will authenticate the current OAuth2 provider as a client for the
// current user. If there is no current user then they will either be looked up
// in the database via the name of the provider, and the ID of the user for that
// provider. If this lookup fails, then a user is created using the information
// from that provider. The password generated for the user will be a random 16
// byte slice, this will never be disclosed to the user, and is there simply
// for security measures.
func (h Oauth2) AuthClient(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	name := mux.Vars(r)["provider"]

	prv, ok := h.Providers[name]

	if !ok {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	u, _ := user.FromContext(r.Context())

	q := r.URL.Query()

	if q.Get("state") != string(prv.Secret()) {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	access, refresh, providerUser, err := prv.Auth(r.Context(), q.Get("code"))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to connect to "+name), "alert")
		h.Redirect(w, r, "/settings")
		return
	}

	// If the user is not logged in, then try and find them in the database,
	// otherwise create the user using the information about them from the
	// provider they just authenticated against.
	if u.IsZero() {
		u, err = h.Users.Get(
			query.WhereQuery("id", "=", provider.Select(
				"user_id",
				query.Where("provider_user_id", "=", providerUser.ID),
				query.Where("name", "=", name),
			)),
			query.OrWhere("email", "=", providerUser.Email),
		)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to connect to "+name), "alert")
			h.RedirectBack(w, r)
			return
		}

		if u.IsZero() {
			password := make([]byte, 16)

			if _, err := rand.Read(password); err != nil {
				h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
				sess.AddFlash(template.Danger("Failed to connect to "+name), "alert")
				h.Redirect(w, r, "/settings")
				return
			}

			username := providerUser.Username

			if username == "" {
				username = providerUser.Login
			}

			u, err = h.Users.Create(providerUser.Email, username, password)

			if err != nil {
				h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
				sess.AddFlash(template.Danger("Failed to connect to "+name), "alert")
				h.Redirect(w, r, "/settings")
				return
			}
		}

		encoded, err := h.SecureCookie.Encode("user", strconv.FormatInt(u.ID, 10))

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to connect to "+name), "alert")
			h.Redirect(w, r, "/settings")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "user",
			HttpOnly: true,
			MaxAge:   user.MaxAge,
			Expires:  time.Now().Add(time.Duration(user.MaxAge) * time.Second),
			Value:    encoded,
			Path:     "/",
		})
	}

	providers := provider.NewStore(h.DB, u)

	p, err := providers.Get(query.Where("name", "=", name))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to connect to "+name), "alert")
		h.Redirect(w, r, "/settings")
		return
	}

	if p.IsZero() {
		p, err = providers.Create(providerUser.ID, name, access, refresh, true)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to connect to "+name), "alert")
			h.Redirect(w, r, "/settings")
			return
		}
	} else {
		if err := providers.Update(p.ID, providerUser.ID, name, access, refresh, true); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to connect to "+name), "alert")
			h.Redirect(w, r, "/settings")
			return
		}
	}

	sess.AddFlash(template.Success("Successfully connected to "+name), "alert")
	h.Redirect(w, r, "/settings")
}

func (h Oauth2) RevokeClient(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	name := mux.Vars(r)["provider"]

	if _, ok := h.Providers[name]; !ok {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	providers := provider.NewStore(h.DB, u)

	p, err := providers.Get(query.Where("name", "=", name))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to disconnect from provider"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if p.IsZero() {
		h.RedirectBack(w, r)
		return
	}

	if err := providers.Update(p.ID, 0, p.Name, nil, nil, false); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to disconnect from provider"), "alert")
		h.RedirectBack(w, r)
		return
	}
	sess.AddFlash(template.Success("Successfully disconnected from provider"), "alert")
	h.RedirectBack(w, r)
}
