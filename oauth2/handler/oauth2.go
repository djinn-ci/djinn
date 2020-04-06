package handler

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
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

	Apps      oauth2.AppStore
	Providers map[string]oauth2.Provider
}

func (h Oauth2) handleAuthPage(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u := h.User(r)
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
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	// User is logged in, so check for an existing token and update the scope
	// for the token if a new scope is requested.
	if !u.IsZero() {
		t, err := oauth2.NewTokenStore(h.DB, u, a).Get()

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if !t.IsZero() {
			diff := oauth2.ScopeDiff(scope, t.Scope)

			// New scope has been requested, so authorize that new scope.
			if len(diff) > 0 {
				goto render
			}

			scope = append(scope, diff...)
			codes := oauth2.NewCodeStore(h.DB, u)

			c := codes.New()
			c.Code = make([]byte, 16)
			c.Scope = scope
			c.ExpiresAt = time.Now().Add(time.Minute*10)

			if len(c.Scope) == 0 {
				c.Scope = t.Scope
			}

			if _, err := rand.Read(c.Code); err != nil {
				log.Error.Println(errors.Err(err))
				web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
				return
			}

			if err := codes.Create(c); err != nil {
				log.Error.Println(errors.Err(err))
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

render:
	p := &oauth2template.Auth{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.FormErrors(sess),
			Fields: h.FormFields(sess),
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

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); ok {
			h.RedirectBack(w, r)
			return
		}
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	var (
		u   *user.User = h.User(r)
		err error
	)

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
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if f.Authenticate {
		u, err = h.Users.Auth(f.Handle, f.Password)

		if err != nil {
			if errors.Cause(err) != user.ErrAuth {
				log.Error.Println(errors.Err(err))
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

	codes := oauth2.NewCodeStore(h.DB, u, a)

	c := codes.New()
	c.Code = make([]byte, 16)
	c.Scope = scope
	c.ExpiresAt = time.Now().Add(time.Minute*10)

	if _, err := rand.Read(c.Code); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := codes.Create(c); err != nil {
		log.Error.Println(errors.Err(err))
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

	clientId, err := hex.DecodeString(q.Get("client_id"))

	if err != nil {
		web.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	clientSecret, err := hex.DecodeString(q.Get("client_secret"))

	if err != nil {
		web.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	code, err := hex.DecodeString(q.Get("code"))

	if err != nil {
		web.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	a, err := h.Apps.Auth(clientId, clientSecret)

	if err != nil {
		if errors.Cause(err) == oauth2.ErrAuth {
			web.JSONError(w, "invalid client id and secret", http.StatusBadRequest)
			return
		}
		log.Error.Println(errors.Err(err))
		web.JSONError(w, errors.Cause(err).Error(), http.StatusInternalServerError)
		return
	}

	codes := oauth2.NewCodeStore(h.DB, a)
	c, err := codes.Get(query.Where("code", "=", code))

	if err != nil {
		log.Error.Println(errors.Err(err))
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
		log.Error.Println(errors.Err(err))
		web.JSONError(w, errors.Cause(err).Error(), http.StatusInternalServerError)
		return
	}

	tokens := oauth2.NewTokenStore(h.DB, a, u)

	t, err := tokens.Get()

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.JSONError(w, errors.Cause(err).Error(), http.StatusInternalServerError)
		return
	}

	if t.IsZero() {
		t = tokens.New()
	}

	t.Token = make([]byte, 16)
	t.Scope = c.Scope

	if _, err := rand.Read(t.Token); err != nil {
		log.Error.Println(errors.Err(err))
		web.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fn := tokens.Update

	if t.ID == 0 {
		fn = tokens.Create
	}

	if err := fn(t); err != nil {
		log.Error.Println(errors.Err(err))
		web.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := codes.Delete(c); err != nil {
		log.Error.Println(errors.Err(err))
		web.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body := map[string]string{
		"access_token": hex.EncodeToString(t.Token),
		"token_type":  "bearer",
		"scope":       t.Scope.String(),
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
		log.Error.Println(errors.Err(err))
		web.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if t.IsZero() {
		goto resp
	}

	if err := h.Tokens.Delete(t); err != nil {
		log.Error.Println(errors.Err(err))
		web.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
resp:
	w.WriteHeader(http.StatusNoContent)
}

func (h Oauth2) AuthClient(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	name := mux.Vars(r)["provider"]

	prv, ok := h.Providers[name]

	if !ok {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	u := h.User(r)
	q := r.URL.Query()

	if q.Get("state") != string(prv.Secret()) {
		web.Text(w, "Not found", http.StatusNotFound)
		return
	}

	access, refresh, userId, err := prv.Auth(r.Context(), q.Get("code"))

	if err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to connect to "+name), "alert")
		h.RedirectBack(w, r)
		return
	}

	providers := provider.NewStore(h.DB, u)

	p, err := providers.Get(query.Where("name", "=", name))

	if err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to connect to "+name), "alert")
		h.RedirectBack(w, r)
		return
	}

	if p.IsZero() {
		p = providers.New()
	}

	p.ProviderUserID = sql.NullInt64{
		Int64: userId,
		Valid: true,
	}
	p.Name = name
	p.AccessToken = access
	p.RefreshToken = refresh
	p.Connected = true

	fn := providers.Update

	if p.ID == 0 {
		fn = providers.Create
	}

	if err := fn(p); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to connect to "+name), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Successfully connected to "+name), "alert")
	h.RedirectBack(w, r)
}

func (h Oauth2) RevokeClient(w http.ResponseWriter, r *http.Request) {

}
