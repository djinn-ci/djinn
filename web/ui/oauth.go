package ui

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/model/types"
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/app"
	"github.com/andrewpillar/thrall/template/user"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"golang.org/x/crypto/bcrypt"
)

type Oauth struct {
	web.Handler

	Apps      model.AppStore
	Providers map[string]oauth2.Provider
}

func (h Oauth) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)
	defer save(r, w)

	u := h.User(r)

	aa, err := u.AppStore().All()

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}

	sp := user.SettingsPage{
		BasePage: bp,
		Form:     template.Form{
			Errors: h.FormErrors(sess),
			Fields: h.FormFields(sess),
		},
		CSRF:     string(csrf.TemplateField(r)),
	}

	p := &app.Index{
		SettingsPage: sp,
		Apps:         aa,
	}

	d := template.NewDashboard(p, r.URL, h.Alert(sess), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Oauth) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)
	defer save(r, w)

	p := &app.Create{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.FormErrors(sess),
			Fields: h.FormFields(sess),
		},
	}

	d := template.NewDashboard(p, r.URL, h.Alert(sess), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Oauth) Store(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)
	defer save(r, w)

	u := h.User(r)

	f := &form.App{}

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); !ok {
			log.Error.Println(errors.Err(err))
		}
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	id := make([]byte, 16)
	secret := make([]byte, 32)

	if _, err := rand.Read(id); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	hash, err := bcrypt.GenerateFromPassword(secret, bcrypt.DefaultCost)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if _, err := rand.Read(secret); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	apps := u.AppStore()

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
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	sess.AddFlash(template.Success("Created OAuth App " + a.Name), "alert")
	http.Redirect(w, r, "/settings/apps", http.StatusSeeOther)
}

func (h Oauth) handleAuthPage(w http.ResponseWriter, r *http.Request, sess *sessions.Session) {
	u := h.User(r)

	q := r.URL.Query()

	clientId := q.Get("client_id")
	redirectUri := q.Get("redirect_uri")
	state := q.Get("state")

	scope, err := types.UnmarshalScope(q.Get("scope"))

	if err != nil {
		web.HTMLError(w, errors.Cause(err).Error(), http.StatusBadRequest)
		return
	}

	if clientId == "" {
		web.HTMLError(w, "No client ID in request", http.StatusBadRequest)
		return
	}

	a, err := h.Apps.Get(query.Where("client_id", "=", []byte(clientId)))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	// User is logged in, so check for an existing token, and update the
	// scope for the token if a new scope is request.
	if !u.IsZero() {
		t, err := u.TokenStore().Get(query.Where("app_id", "=", a.ID))

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if !t.IsZero() {
			diff := types.ScopeDiff(scope, t.Scope)

			scope = append(scope, diff...)
		}

		if !t.IsZero() {
			codes := u.CodeStore()
			c := codes.New()
			c.Code = make([]byte, 16)
			c.Scope = scope
			c.ExpiresAt = time.Now().Add(time.Minute*10)

			if len(c.Scope) == 0 {
				c.Scope = t.Scope
			}

			rand.Read(c.Code)

			if err := codes.Create(c); err != nil {
				log.Error.Println(errors.Err(err))
				web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
				return
			}

			redirectQuery := url.Values(make(map[string][]string))
			redirectQuery.Add("code", fmt.Sprintf("%x", c.Code))

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

	p := &app.Auth{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.FormErrors(sess),
			Fields: h.FormFields(sess),
		},
		User:        u,
		Name:        a.Name,
		Scope:       scope,
		ClientID:    a.ClientID,
		RedirectURI: a.RedirectURI,
		State:       state,
	}

	web.HTML(w, template.Render(p), http.StatusOK)
	return
}

func (h Oauth) Auth(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)
	defer save(r, w)

	if r.Method == "GET" {
		h.handleAuthPage(w, r, sess)
		return
	}

	f := &form.AuthorizeApp{}

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); ok {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	var err error

	u := h.User(r)

	if f.Login {
		u, err = h.Users.Auth(f.Handle, f.Password)

		if err != nil {
			if errors.Cause(err) != model.ErrAuth {
				log.Error.Println(errors.Err(err))
				web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
				return
			}

			errs := form.NewErrors()
			errs.Put("handle", model.ErrAuth)
			errs.Put("password", model.ErrAuth)

			sess.AddFlash(errs, "form_errors")
			sess.AddFlash(f.Fields(), "form_fields")

			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}
	}

	codes := u.CodeStore()
	c := codes.New()
	c.Code = make([]byte, 16)
	c.Scope = f.Scope
	c.ExpiresAt = time.Now().Add(time.Minute*10)

	redirectQuery := url.Values(make(map[string][]string))
	redirectQuery.Add("code", fmt.Sprintf("%x", c.Code))

	if f.State != "" {
		redirectQuery.Add("state", f.State)
	}

	http.Redirect(w, r, f.RedirectURI+"?"+redirectQuery.Encode(), http.StatusSeeOther)
}

func (h Oauth) Token(w http.ResponseWriter, r *http.Request) {

}

func (h Oauth) AuthClient(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)
	defer save(r, w)

	name := mux.Vars(r)["provider"]

	provider, ok := h.Providers[name]

	if !ok {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	u := h.User(r)

	if r.URL.Query().Get("state") != string(provider.Secret()) {
		web.Text(w, "Not found", http.StatusNotFound)
		return
	}

	if err := provider.Auth(r.Context(), r.URL.Query().Get("code"), u.ProviderStore()); err != nil {
		log.Error.Println(errors.Err(err))

		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	sess.AddFlash(template.Success("Successfully connected to " + name), "alert")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h Oauth) RevokeClient(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)
	defer save(r, w)

	name := mux.Vars(r)["provider"]

	provider, ok := h.Providers[name]

	if !ok {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	u := h.User(r)

	providers := u.ProviderStore()

	p, err := providers.Get(query.Where("name", "=", name))

	if err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		sess.AddFlash(template.Danger("Failed to revoke provider: " + cause.Error()), "alert")
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if err := provider.Revoke(p); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		sess.AddFlash(template.Danger("Failed to revoke provider: " + cause.Error()), "alert")
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	p.Connected = false
	p.AccessToken = nil
	p.RefreshToken = nil

	if err := providers.Update(p); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		sess.AddFlash(template.Danger("Failed to revoke provider: " + cause.Error()), "alert")
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	sess.AddFlash(template.Success("Successfully revoked access to: " + name), "alert")
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
