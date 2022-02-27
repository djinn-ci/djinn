package http

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/alert"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/oauth2"
	oauth2template "djinn-ci.com/oauth2/template"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"
	usertemplate "djinn-ci.com/user/template"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type Handler struct {
	*server.Server

	Apps   *oauth2.AppStore
	Tokens oauth2.TokenStore
	Codes  oauth2.CodeStore
	Users  user.Store
	User   *userhttp.Handler
}

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server: srv,
		Apps: &oauth2.AppStore{
			Pool:   srv.DB,
			AESGCM: srv.AESGCM,
		},
		Tokens: oauth2.TokenStore{Pool: srv.DB},
		Codes:  oauth2.CodeStore{Pool: srv.DB},
		Users:  user.Store{Pool: srv.DB},
		User:   userhttp.NewHandler(srv),
	}
}

type Oauth2 struct {
	*Handler
}

func (h Oauth2) authPage(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, loggedIn, err := h.User.UserFromRequest(r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	q := r.URL.Query()

	clientId := q.Get("client_id")
	redirectUri := q.Get("redirect_uri")
	state := q.Get("state")

	if clientId == "" {
		h.Error(w, r, "No client ID in request", http.StatusBadRequest)
		return
	}

	scope, err := oauth2.UnmarshalScope(q.Get("scope"))

	if err != nil {
		h.Error(w, r, errors.Cause(err).Error(), http.StatusBadRequest)
		return
	}

	a, ok, err := h.Apps.Get(query.Where("client_id", "=", query.Arg(clientId)))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	author, _, err := h.Users.Get(query.Where("id", "=", query.Arg(a.UserID)))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if loggedIn {
		tok, ok, err := h.Tokens.Get(
			query.Where("user_id", "=", query.Arg(u.ID)),
			query.Where("app_id", "=", query.Arg(a.ID)),
		)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if ok {
			diff := oauth2.ScopeDiff(scope, tok.Scope)

			if len(diff) == 0 {
				if a.RedirectURI != redirectUri {
					h.Error(w, r, "redirect_uri does not match", http.StatusBadRequest)
					return
				}

				c, err := h.Codes.Create(oauth2.CodeParams{
					UserID: u.ID,
					AppID:  a.ID,
					Scope:  scope,
				})

				if err != nil {
					h.InternalServerError(w, r, errors.Err(err))
					return
				}

				redirectQuery := make(url.Values)
				redirectQuery.Add("code", c.Code)

				if state != "" {
					redirectQuery.Add("state", state)
				}

				http.Redirect(w, r, redirectUri+"?"+redirectQuery.Encode(), http.StatusSeeOther)
				return
			}
			scope = append(scope, diff...)
		}
	}

	if len(scope) == 0 {
		h.Error(w, r, "No scope requested", http.StatusBadRequest)
		return
	}

	p := &oauth2template.Auth{
		Form: template.Form{
			CSRF:   csrf.TemplateField(r),
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
		User:        u,
		Author:      author,
		Name:        a.Name,
		RedirectURI: redirectUri,
		State:       state,
		Scope:       scope,
	}
	save(r, w)
	webutil.HTML(w, template.Render(p), http.StatusOK)
}

func (h Oauth2) Auth(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		h.authPage(w, r)
		return
	}

	sess, _ := h.Session(r)

	u, loggedIn, err := h.User.UserFromRequest(r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	var f AuthorizeForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	v := AuthorizeValidator{
		Form: f,
	}

	if err := webutil.Validate(&v); err != nil {
		verrs := err.(webutil.ValidationErrors)
		webutil.FlashFormWithErrors(sess, f, verrs)
		h.RedirectBack(w, r)
		return
	}

	a, ok, err := h.Apps.Get(query.Where("client_id", "=", query.Arg(f.ClientID)))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	if f.RedirectURI != a.RedirectURI {
		h.Error(w, r, "redirect_uri does not match", http.StatusBadRequest)
		return
	}

	if !loggedIn {
		u, err = h.Users.Auth(f.Handle, f.Password)

		if err != nil {
			if !errors.Is(err, user.ErrAuth) {
				h.InternalServerError(w, r, errors.Err(err))
				return
			}

			errs := webutil.NewValidationErrors()
			errs.Add("handle", user.ErrAuth)
			errs.Add("password", user.ErrAuth)

			sess.AddFlash(errs, "form_errors")
			sess.AddFlash(f.Fields(), "form_fields")
			h.RedirectBack(w, r)
			return
		}
	}

	scope, err := oauth2.UnmarshalScope(f.Scope)

	if err != nil {
		h.Error(w, r, "Invalid scope", http.StatusBadRequest)
		return
	}

	c, err := h.Codes.Create(oauth2.CodeParams{
		UserID: u.ID,
		AppID:  a.ID,
		Scope:  scope,
	})

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	q := make(url.Values)
	q.Add("code", c.Code)

	if f.State != "" {
		q.Add("state", f.State)
	}
	http.Redirect(w, r, f.RedirectURI+"?"+q.Encode(), http.StatusSeeOther)
}

func (h Oauth2) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	id := r.Form.Get("client_id")
	secret := r.Form.Get("secret")
	code := r.Form.Get("code")

	if auth := r.Header.Get("Authorization"); auth != "" {
		var ok bool

		id, secret, ok = r.BasicAuth()

		if !ok {
			h.Error(w, r, "Could not get basic auth credentials", http.StatusBadRequest)
			return
		}
	}

	a, err := h.Apps.Auth(id, secret)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	c, ok, err := h.Codes.Get(
		query.Where("app_id", "=", query.Arg(a.ID)),
		query.Where("code", "=", query.Arg(code)),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	if time.Now().Sub(c.ExpiresAt) > time.Minute*10 {
		h.Error(w, r, "Code expired", http.StatusBadRequest)
		return
	}

	t, ok, err := h.Tokens.Get(
		query.Where("user_id", "=", query.Arg(c.UserID)),
		query.Where("app_id", "=", query.Arg(c.AppID)),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	p := oauth2.TokenParams{
		UserID: c.UserID,
		AppID:  c.AppID,
		Name:   "client." + strconv.FormatInt(c.UserID, 10),
		Scope:  c.Scope,
	}

	if !ok {
		t, err = h.Tokens.Create(p)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}
	} else {
		if err := h.Tokens.Update(t.ID, p); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}
	}

	if err := h.Codes.Delete(c.ID); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	body := map[string]string{
		"access_token": t.Token,
		"token_type":   "bearer",
		"scope":        t.Scope.String(),
	}

	if server.IsJSON(r) {
		webutil.JSON(w, body, http.StatusOK)
		return
	}

	vals := make(url.Values)

	for k, v := range body {
		vals.Add(k, v)
	}
	webutil.Text(w, vals.Encode(), http.StatusOK)
}

func (h Oauth2) Revoke(w http.ResponseWriter, r *http.Request) {
	prefix := "Bearer "

	tok := r.Header.Get("Authorization")
	tok = tok[len(prefix):]

	t, ok, err := h.Tokens.Get(query.Where("token", "=", query.Arg(tok)))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		if err := h.Tokens.Delete(t.ID); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

type App struct {
	*Handler
}

type AppHandlerFunc func(*user.User, *oauth2.App, http.ResponseWriter, *http.Request)

func (h App) WithApp(fn AppHandlerFunc) userhttp.HandlerFunc {
	return func(u *user.User, w http.ResponseWriter, r *http.Request) {
		clientId := mux.Vars(r)["app"]

		a, ok, err := h.Apps.Get(
			query.Where("user_id", "=", query.Arg(u.ID)),
			query.Where("client_id", "=", query.Arg(clientId)),
		)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}
		fn(u, a, w, r)
	}
}

func (h App) Index(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	aa, err := h.Apps.All(query.Where("user_id", "=", query.Arg(u.ID)))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	csrf := csrf.TemplateField(r)

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}

	p := &usertemplate.Settings{
		BasePage: bp,
		Section: &oauth2template.AppIndex{
			BasePage: bp,
			Apps:     aa,
		},
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h App) Create(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrf := csrf.TemplateField(r)

	p := &oauth2template.AppForm{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h App) Store(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	var f AppForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to create app")
		h.RedirectBack(w, r)
		return
	}

	v := AppValidator{
		UserID: u.ID,
		Form:   f,
		Apps:   h.Apps,
	}

	if err := webutil.Validate(v); err != nil {
		verrs := err.(webutil.ValidationErrors)

		if errs, ok := verrs["fatal"]; ok {
			h.Log.Error.Println(r.Method, r.URL, errors.Slice(errs))
			alert.Flash(sess, alert.Danger, "Failed to submit build")
			h.RedirectBack(w, r)
			return
		}

		webutil.FlashFormWithErrors(sess, f, verrs)
		h.RedirectBack(w, r)
		return
	}

	a, err := h.Apps.Create(oauth2.AppParams{
		UserID:      u.ID,
		Name:        f.Name,
		Description: f.Description,
		HomeURI:     f.HomepageURI,
		RedirectURI: f.RedirectURI,
	})

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to create app")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Create OAuth app "+a.Name)
	h.Redirect(w, r, "/settings/apps")
}

func (h App) Show(u *user.User, a *oauth2.App, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	b, err := h.AESGCM.Decrypt(a.ClientSecret)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	a.ClientSecret = b

	csrf := csrf.TemplateField(r)

	p := &oauth2template.AppForm{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
		App: a,
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h App) Update(u *user.User, a *oauth2.App, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	base := webutil.BasePath(r.URL.Path)

	if base == "revoke" {
		if err := h.Tokens.Revoke(a.ID); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to revoke tokens")
			h.RedirectBack(w, r)
			return
		}

		alert.Flash(sess, alert.Success, "App tokens revoked")
		h.RedirectBack(w, r)
		return
	}

	if base == "reset" {
		if err := h.Apps.Reset(a.ID); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to reset secret")
			h.RedirectBack(w, r)
			return
		}

		alert.Flash(sess, alert.Success, "App client secret reset")
		h.RedirectBack(w, r)
		return
	}

	var f AppForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to update app")
		h.RedirectBack(w, r)
		return
	}

	v := AppValidator{
		UserID: u.ID,
		Form:   f,
		Apps:   h.Apps,
		App:    a,
	}

	if err := webutil.Validate(v); err != nil {
		verrs := err.(webutil.ValidationErrors)
		if errs, ok := verrs["fatal"]; ok {
			h.Log.Error.Println(r.Method, r.URL, errors.Slice(errs))
			alert.Flash(sess, alert.Danger, "Failed to submit build")
			h.RedirectBack(w, r)
			return
		}

		webutil.FlashFormWithErrors(sess, f, verrs)
		h.RedirectBack(w, r)
		return
	}

	params := oauth2.AppParams{
		UserID:      u.ID,
		Name:        f.Name,
		Description: f.Description,
		HomeURI:     f.HomepageURI,
		RedirectURI: f.RedirectURI,
	}

	if err := h.Apps.Update(a.ID, params); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to update app")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "App channges have been save")
	h.Redirect(w, r, a.Endpoint())
}

type Token struct {
	*Handler
}

type TokenHandlerFunc func(*user.User, *oauth2.Token, http.ResponseWriter, *http.Request)

func (h Token) WithToken(fn TokenHandlerFunc) userhttp.HandlerFunc {
	return func(u *user.User, w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		opt := query.Where("id", "=", query.Arg(vars["token"]))

		if id, ok := vars["client_id"]; ok {
			a, ok, err := h.Apps.Get(query.Where("client_id", "=", query.Arg(id)))

			if err != nil {
				h.InternalServerError(w, r, errors.Err(err))
				return
			}

			if !ok {
				h.NotFound(w, r)
				return
			}
			opt = query.Where("app_id", "=", query.Arg(a.ID))
		}

		t, ok, err := h.Tokens.Get(opt)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}
		fn(u, t, w, r)
	}
}

func (h Token) Index(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	var (
		tt  []*oauth2.Token
		err error

		section template.Section
	)

	csrf := csrf.TemplateField(r)

	switch webutil.BasePath(r.URL.Path) {
	case "connections":
		tt, err = h.Tokens.All(
			query.Where("user_id", "=", query.Arg(u.ID)),
			query.Where("app_id", "IS NOT", query.Lit("NULL")),
			query.OrderDesc("created_at"),
		)

		section = &oauth2template.ConnectionIndex{
			BasePage: template.BasePage{
				URL:  r.URL,
				User: u,
			},
			Tokens: tt,
		}
	case "tokens":
		tt, err = h.Tokens.All(
			query.Where("user_id", "=", query.Arg(u.ID)),
			query.Where("app_id", "IS", query.Lit("NULL")),
			query.OrderDesc("created_at"),
		)

		section = &oauth2template.TokenIndex{
			CSRF:   csrf,
			Tokens: tt,
		}
	}

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	flashes := sess.Flashes("token_id")

	mm := make([]database.Model, 0, len(tt))

	for _, t := range tt {
		mm = append(mm, t)

		if len(flashes) > 0 {
			if id, _ := flashes[0].(int64); id == t.ID {
				continue
			}
		}
		t.Token = ""
	}

	if err := h.Apps.Load("app_id", "id", mm...); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	mm = mm[0:0]

	for _, t := range tt {
		if t.App != nil {
			mm = append(mm, t.App)
		}
	}

	if err := h.Users.Load("user_id", "id", mm...); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	p := &usertemplate.Settings{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Section: section,
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h Token) Create(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrf := csrf.TemplateField(r)
	fields := webutil.FormFields(sess)

	p := &oauth2template.TokenForm{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: fields,
		},
		Scopes: make(map[string]struct{}),
	}

	scope := strings.Split(fields["scope"], " ")

	for _, sc := range scope {
		p.Scopes[sc] = struct{}{}
	}

	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h Token) Store(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	var f TokenForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	v := TokenValidator{
		UserID: u.ID,
		Form:   f,
		Tokens: h.Tokens,
	}

	if err := webutil.Validate(v); err != nil {
		verrs := err.(webutil.ValidationErrors)
		if errs, ok := verrs["fatal"]; ok {
			h.Log.Error.Println(r.Method, r.URL, errors.Slice(errs))
			alert.Flash(sess, alert.Danger, "Failed to submit build")
			h.RedirectBack(w, r)
			return
		}

		webutil.FlashFormWithErrors(sess, f, verrs)
		h.RedirectBack(w, r)
		return
	}

	tok, err := h.Tokens.Create(oauth2.TokenParams{
		UserID: u.ID,
		Name:   f.Name,
		Scope:  f.Scope,
	})

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	sess.AddFlash(tok.ID, "token_id")
	h.Redirect(w, r, "/settings/tokens")
}

func (h Token) Show(u *user.User, t *oauth2.Token, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrf := csrf.TemplateField(r)

	parts := strings.Split(r.URL.Path, "/")

	if parts[len(parts)-2] == "connections" {
		p := &oauth2template.Connection{
			BasePage: template.BasePage{
				URL:  r.URL,
				User: u,
			},
			CSRF:  csrf,
			Token: t,
		}
		d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
		save(r, w)
		webutil.HTML(w, template.Render(d), http.StatusOK)
		return
	}

	p := &oauth2template.TokenForm{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
		Token:  t,
		Scopes: t.Permissions(),
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h Token) Update(u *user.User, t *oauth2.Token, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if webutil.BasePath(r.URL.Path) == "regenerate" {
		if err := h.Tokens.Reset(t.ID); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to update token")
			h.RedirectBack(w, r)
			return
		}

		sess.AddFlash(t.ID, "token_id")
		alert.Flash(sess, alert.Success, "Token has been updated: "+t.Name)
		h.Redirect(w, r, "/settings/tokens")
		return
	}

	var f TokenForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	v := TokenValidator{
		UserID: u.ID,
		Form:   f,
		Tokens: h.Tokens,
		Token:  t,
	}

	if err := webutil.Validate(v); err != nil {
		verrs := err.(webutil.ValidationErrors)
		if errs, ok := verrs["fatal"]; ok {
			h.Log.Error.Println(r.Method, r.URL, errors.Slice(errs))
			alert.Flash(sess, alert.Danger, "Failed to submit build")
			h.RedirectBack(w, r)
			return
		}

		webutil.FlashFormWithErrors(sess, f, verrs)
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(t.ID, "token_id")
	alert.Flash(sess, alert.Success, "Token has been updated: "+t.Name)
	h.Redirect(w, r, "/settings/tokens")
}

func (h Token) Destroy(u *user.User, t *oauth2.Token, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.Tokens.Delete(t.ID); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to disconnect from app")
		h.RedirectBack(w, r)
		return
	}

	parts := strings.Split(r.URL.Path, "/")

	if parts[len(parts)-2] == "connections" {
		alert.Flash(sess, alert.Success, "Successfully disconnected from app")
		h.Redirect(w, r, "/settings/connections")
		return
	}

	alert.Flash(sess, alert.Success, "Token deleted")
	h.Redirect(w, r, "/settings/tokens")
}

func RegisterUI(srv *server.Server) {
	h := NewHandler(srv)

	oauth := Oauth2{
		Handler: h,
	}

	srv.Router.HandleFunc("/login/oauth/authorize", oauth.Auth).Methods("GET", "POST")
	srv.Router.HandleFunc("/login/oauth/token", oauth.Token).Methods("POST")
	srv.Router.HandleFunc("/logout/oauth/revoke", oauth.Revoke).Methods("POST")

	app := App{
		Handler: h,
	}

	tok := Token{
		Handler: h,
	}

	user := userhttp.NewHandler(srv)

	auth := srv.Router.PathPrefix("/settings").Subrouter()
	auth.HandleFunc("/tokens", user.WithUser(tok.Index)).Methods("GET")
	auth.HandleFunc("/tokens/create", user.WithUser(tok.Create)).Methods("GET")
	auth.HandleFunc("/tokens", user.WithUser(tok.Store)).Methods("POST")
	auth.HandleFunc("/tokens/{token}", user.WithUser(tok.WithToken(tok.Show))).Methods("GET")
	auth.HandleFunc("/tokens/{token}", user.WithUser(tok.WithToken(tok.Update))).Methods("PATCH")
	auth.HandleFunc("/tokens/{token}/regenerate", user.WithUser(tok.WithToken(tok.Update))).Methods("PATCH")
	auth.HandleFunc("/tokens/{token}", user.WithUser(tok.WithToken(tok.Destroy))).Methods("DELETE")
	auth.HandleFunc("/tokens/revoke", user.WithUser(tok.WithToken(tok.Destroy))).Methods("DELETE")
	auth.HandleFunc("/apps", user.WithUser(app.Index)).Methods("GET")
	auth.HandleFunc("/apps/create", user.WithUser(app.Create)).Methods("GET")
	auth.HandleFunc("/apps", user.WithUser(app.Store)).Methods("POST")
	auth.HandleFunc("/apps/{app}", user.WithUser(app.WithApp(app.Show))).Methods("GET")
	auth.HandleFunc("/apps/{app}", user.WithUser(app.WithApp(app.Update))).Methods("PATCH")
	auth.HandleFunc("/apps/{app}/revoke", user.WithUser(app.WithApp(app.Update))).Methods("PATCH")
	auth.HandleFunc("/apps/{app}/reset", user.WithUser(app.WithApp(app.Update))).Methods("PATCH")
	auth.HandleFunc("/connections", user.WithUser(tok.Index)).Methods("GET")
	auth.HandleFunc("/connections/{id}", user.WithUser(tok.WithToken(tok.Show))).Methods("GET")
	auth.HandleFunc("/connections/{id}", user.WithUser(tok.WithToken(tok.Destroy))).Methods("DELETE")
	auth.Use(srv.CSRF)
}
