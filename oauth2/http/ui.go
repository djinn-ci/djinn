package http

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/alert"
	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/oauth2"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/template/form"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"

	"github.com/gorilla/mux"
)

type Handler struct {
	*server.Server

	Auth   auth.Authenticator
	Apps   *oauth2.AppStore
	Tokens oauth2.TokenStore
	Codes  oauth2.CodeStore
	Users  *database.Store[*auth.User]
}

func NewHandler(a auth.Authenticator, srv *server.Server) *Handler {
	return &Handler{
		Server: srv,
		Auth:   a,
		Apps: &oauth2.AppStore{
			Store:  oauth2.NewAppStore(srv.DB),
			AESGCM: srv.AESGCM,
		},
		Tokens: oauth2.TokenStore{
			Store: oauth2.NewTokenStore(srv.DB),
		},
		Codes: oauth2.CodeStore{
			Store: oauth2.NewCodeStore(srv.DB),
		},
		Users: user.NewStore(srv.DB),
	}
}

type Oauth2 struct {
	*Handler
}

func (h Oauth2) authPage(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u, err := h.Auth.Auth(r)

	if err != nil {
		if !errors.Is(err, auth.ErrAuth) {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to authenticate request"))
			return
		}
	}

	q := r.URL.Query()

	clientId := q.Get("client_id")
	redirectUri := q.Get("redirect_uri")
	state := q.Get("state")

	if clientId == "" {
		h.InternalServerError(w, r, errors.Benign("No client ID in request"))
		return
	}

	scope, err := oauth2.UnmarshalScope(q.Get("scope"))

	if err != nil {
		h.InternalServerError(w, r, errors.Cause(err))
		return
	}

	ctx := r.Context()

	a, ok, err := h.Apps.Get(ctx, query.Where("client_id", "=", query.Arg(clientId)))

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get app"))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	author, _, err := h.Users.Get(ctx, user.WhereID(a.UserID))

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get user"))
		return
	}

	if u.ID > 0 {
		tok, ok, err := h.Tokens.Get(
			ctx,
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
					h.InternalServerError(w, r, errors.Benign("redirect_uri does not match"))
					return
				}

				c, err := h.Codes.Create(ctx, &oauth2.CodeParams{
					User:  u,
					AppID: a.ID,
					Scope: scope,
				})

				if err != nil {
					h.InternalServerError(w, r, errors.Wrap(err, "Failed to create authentication code"))
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
		h.InternalServerError(w, r, errors.Wrap(err, "No scope requested"))
		return
	}

	tmpl := template.Oauth2Login{
		Form:        form.New(sess, r),
		User:        u,
		Author:      author,
		RedirectURI: redirectUri,
		State:       state,
		Scope:       scope,
	}
	h.Template(w, r, &tmpl, http.StatusOK)
}

func (h Oauth2) Authz(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		h.authPage(w, r)
		return
	}

	u, err := h.Auth.Auth(r)

	if err != nil {
		if errors.Is(err, auth.ErrAuth) {
			h.FormError(w, r, nil, errors.Benign("Invalid credentials"))
			return
		}

		h.InternalServerError(w, r, errors.Wrap(err, "Failed to authenticate request"))
		return
	}

	var f AuthorizeForm

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to authenticate request"))
		return
	}

	ctx := r.Context()

	a, ok, err := h.Apps.Get(ctx, query.Where("client_id", "=", query.Arg(f.ClientID)))

	if err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to get app"))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	if f.RedirectURI != a.RedirectURI {
		h.FormError(w, r, &f, errors.Benign("redirect_uri does not match"))
		return
	}

	scope, err := oauth2.UnmarshalScope(f.Scope)

	if err != nil {
		h.FormError(w, r, &f, errors.Benign("Invalid scope"))
		return
	}

	c, err := h.Codes.Create(ctx, &oauth2.CodeParams{
		User:  u,
		AppID: a.ID,
		Scope: scope,
	})

	if err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to generate authentication code"))
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
			h.InternalServerError(w, r, errors.Benign("Could not get basic auth credentials"))
			return
		}
	}

	ctx := r.Context()

	a, err := h.Apps.Auth(ctx, id, secret)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to authenticate"))
		return
	}

	c, ok, err := h.Codes.Get(
		ctx,
		query.Where("app_id", "=", query.Arg(a.ID)),
		query.Where("code", "=", query.Arg(code)),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get code"))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	if time.Now().Sub(c.ExpiresAt) > time.Minute*10 {
		h.InternalServerError(w, r, errors.Benign("Code expired"))
		return
	}

	t, ok, err := h.Tokens.Get(
		ctx,
		query.Where("user_id", "=", query.Arg(c.UserID)),
		query.Where("app_id", "=", query.Arg(c.AppID)),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get token"))
		return
	}

	if !ok {
		t, err = h.Tokens.Create(ctx, &oauth2.TokenParams{
			User:  &auth.User{ID: c.UserID},
			AppID: c.AppID,
			Name:  "client." + strconv.FormatInt(c.UserID, 10),
			Scope: c.Scope,
		})

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to create token"))
			return
		}
	} else {
		t.UserID = c.UserID
		t.AppID.Elem = c.AppID
		t.AppID.Valid = c.AppID > 0
		t.Name = "client." + strconv.FormatInt(c.UserID, 10)
		t.Scope = c.Scope

		if err := h.Tokens.Update(ctx, t); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to update token"))
			return
		}
	}

	if err := h.Codes.Delete(ctx, c); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to delete code"))
		return
	}

	body := map[string]string{
		"access_token": t.Token,
		"token_type":   "bearer",
		"scope":        t.Scope.String(),
	}

	if r.Header.Get("Accept") == "application/json" {
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

	ctx := r.Context()

	t, ok, err := h.Tokens.Get(ctx, query.Where("token", "=", query.Arg(tok)))

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get token"))
		return
	}

	if !ok {
		if err := h.Tokens.Delete(ctx, t); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to delete token"))
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

type App struct {
	*Handler
}

type AppHandlerFunc func(*auth.User, *oauth2.App, http.ResponseWriter, *http.Request)

func (h App) App(fn AppHandlerFunc) auth.HandlerFunc {
	return func(u *auth.User, w http.ResponseWriter, r *http.Request) {
		clientId := mux.Vars(r)["app"]

		a, ok, err := h.Apps.Get(
			r.Context(),
			query.Where("user_id", "=", query.Arg(u.ID)),
			query.Where("client_id", "=", query.Arg(clientId)),
		)

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get app"))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}
		fn(u, a, w, r)
	}
}

func (h App) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	aa, err := h.Apps.All(r.Context(), query.Where("user_id", "=", query.Arg(u.ID)))

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get apps"))
		return
	}

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.Settings{
		Page: tmpl.Page,
		Partial: &template.AppIndex{
			Page: tmpl.Page,
			Apps: aa,
		},
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h App) Create(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.AppForm{
		Form: form.New(sess, r),
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h App) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	f := AppForm{
		Pool: h.DB,
		User: u,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to create app"))
		return
	}

	a, err := h.Apps.Create(r.Context(), &oauth2.AppParams{
		User:        u,
		Name:        f.Name,
		Description: f.Description,
		HomeURI:     f.HomepageURI,
		RedirectURI: f.RedirectURI,
	})

	if err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to create app"))
		return
	}

	alert.Flash(sess, alert.Success, "Create OAuth app "+a.Name)
	h.Redirect(w, r, "/settings/apps")
}

func (h App) Show(u *auth.User, a *oauth2.App, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	b, err := h.AESGCM.Decrypt(a.ClientSecret)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to decrypt client secret"))
		return
	}

	a.ClientSecret = b

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.AppForm{
		Form: form.New(sess, r),
		App:  a,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h App) Update(u *auth.User, a *oauth2.App, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, _ := h.Session(r)

	switch webutil.BasePath(r.URL.Path) {
	case "revoke":
		tok := oauth2.Token{
			AppID: database.Null[int64]{
				Elem:  a.ID,
				Valid: true,
			},
		}

		if err := h.Tokens.Revoke(ctx, &tok); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to revoke tokens"))
			return
		}

		alert.Flash(sess, alert.Success, "App tokens revoked")
		h.RedirectBack(w, r)
		return
	case "reset":
		if err := h.Apps.Reset(ctx, a); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to revoke secret"))
			return
		}

		alert.Flash(sess, alert.Success, "App client secret reset")
		h.RedirectBack(w, r)
		return
	}

	f := AppForm{
		Pool: h.DB,
		App:  a,
		User: u,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to update app"))
		return
	}

	a.Name = f.Name
	a.Description = f.Description
	a.HomeURI = f.HomepageURI
	a.RedirectURI = f.RedirectURI

	if err := h.Apps.Update(ctx, a); err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to update app"))
		return
	}

	alert.Flash(sess, alert.Success, "App channges have been save")
	h.Redirect(w, r, a.Endpoint())
}

type Token struct {
	*Handler
}

type TokenHandlerFunc func(*auth.User, *oauth2.Token, http.ResponseWriter, *http.Request)

func (h Token) Token(fn TokenHandlerFunc) auth.HandlerFunc {
	return func(u *auth.User, w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)

		opt := query.Where("id", "=", query.Arg(vars["token"]))

		if id, ok := vars["client_id"]; ok {
			a, ok, err := h.Apps.Get(ctx, query.Where("client_id", "=", query.Arg(id)))

			if err != nil {
				h.InternalServerError(w, r, errors.Wrap(err, "Failed to get token"))
				return
			}

			if !ok {
				h.NotFound(w, r)
				return
			}
			opt = query.Where("app_id", "=", query.Arg(a.ID))
		}

		t, ok, err := h.Tokens.Get(ctx, opt)

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get token"))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}
		fn(u, t, w, r)
	}
}

func (h Token) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	ctx := r.Context()

	var (
		tt      []*oauth2.Token
		err     error
		partial template.Partial
	)

	tmpl := template.NewDashboard(u, sess, r)

	switch webutil.BasePath(r.URL.Path) {
	case "connections":
		tt, err = h.Tokens.All(
			ctx,
			query.Where("user_id", "=", query.Arg(u.ID)),
			query.Where("app_id", "IS NOT", query.Lit("NULL")),
			query.OrderDesc("created_at"),
		)

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get connected apps"))
			return
		}

		partial = &template.ConnectionIndex{
			Page:   tmpl.Page,
			Tokens: tt,
		}
	case "tokens":
		tt, err = h.Tokens.All(
			ctx,
			query.Where("user_id", "=", query.Arg(u.ID)),
			query.Where("app_id", "IS", query.Lit("NULL")),
			query.OrderDesc("created_at"),
		)

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get tokens"))
			return
		}

		partial = &template.TokenIndex{
			Page:   tmpl.Page,
			Tokens: tt,
		}
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

	if err := oauth2.AppLoader(h.DB).Load(ctx, "app_id", "id", mm...); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to load relations"))
		return
	}

	mm = mm[0:0]

	for _, t := range tt {
		if t.App != nil {
			mm = append(mm, t.App)
		}
	}

	if err := user.Loader(h.DB).Load(ctx, "user_id", "id", mm...); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to load relations"))
		return
	}

	tmpl.Partial = &template.Settings{
		Page:    tmpl.Page,
		Partial: partial,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h Token) Create(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	fields := webutil.FormFields(sess)
	scopes := make(map[string]struct{})

	for _, sc := range strings.Split(fields["scope"], " ") {
		scopes[sc] = struct{}{}
	}

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.TokenForm{
		Form:   form.New(sess, r),
		Scopes: scopes,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h Token) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	f := TokenForm{
		Pool: h.DB,
		User: u,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to create token"))
		return
	}

	tok, err := h.Tokens.Create(r.Context(), &oauth2.TokenParams{
		User:  u,
		Name:  f.Name,
		Scope: f.Scope,
	})

	if err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to create token"))
		return
	}

	sess.AddFlash(tok.ID, "token_id")
	h.Redirect(w, r, "/settings/tokens")
}

func (h Token) Show(u *auth.User, t *oauth2.Token, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	tmpl := template.NewDashboard(u, sess, r)

	parts := strings.Split(r.URL.Path, "/")

	if parts[len(parts)-2] == "connections" {
		tmpl.Partial = &template.ConnectionShow{
			Form:  form.New(sess, r),
			Token: t,
		}
		h.Template(w, r, tmpl, http.StatusOK)
		return
	}

	tmpl.Partial = &template.TokenForm{
		Form:   form.New(sess, r),
		Token:  t,
		Scopes: t.Permissions(),
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h Token) Update(u *auth.User, t *oauth2.Token, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, _ := h.Session(r)

	if webutil.BasePath(r.URL.Path) == "regenerate" {
		if err := h.Tokens.Reset(ctx, t); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to update token"))
			return
		}

		sess.AddFlash(t.ID, "token_id")
		alert.Flash(sess, alert.Success, "Token has been updated: "+t.Name)
		h.Redirect(w, r, "/settings/tokens")
		return
	}

	f := TokenForm{
		Pool:  h.DB,
		Token: t,
		User:  u,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to update token"))
		return
	}

	sess.AddFlash(t.ID, "token_id")
	alert.Flash(sess, alert.Success, "Token has been updated: "+t.Name)
	h.Redirect(w, r, "/settings/tokens")
}

func (h Token) Destroy(u *auth.User, t *oauth2.Token, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.Tokens.Delete(r.Context(), t); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to disconnect from app"))
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

func RegisterUI(a auth.Authenticator, srv *server.Server) {
	h := NewHandler(a, srv)

	oauth := Oauth2{
		Handler: h,
	}

	srv.Router.HandleFunc("/login/oauth/authorize", oauth.Authz).Methods("GET", "POST")
	srv.Router.HandleFunc("/login/oauth/token", oauth.Token).Methods("POST")
	srv.Router.HandleFunc("/logout/oauth/revoke", oauth.Revoke).Methods("POST")

	app := App{
		Handler: h,
	}

	tok := Token{
		Handler: h,
	}

	auth := srv.Router.PathPrefix("/settings").Subrouter()
	auth.HandleFunc("/tokens", srv.Restrict(a, nil, tok.Index)).Methods("GET")
	auth.HandleFunc("/tokens/create", srv.Restrict(a, nil, tok.Create)).Methods("GET")
	auth.HandleFunc("/tokens", srv.Restrict(a, nil, tok.Store)).Methods("POST")
	auth.HandleFunc("/tokens/{token}", srv.Restrict(a, nil, tok.Token(tok.Show))).Methods("GET")
	auth.HandleFunc("/tokens/{token}", srv.Restrict(a, nil, tok.Token(tok.Update))).Methods("PATCH")
	auth.HandleFunc("/tokens/{token}/regenerate", srv.Restrict(a, nil, tok.Token(tok.Update))).Methods("PATCH")
	auth.HandleFunc("/tokens/{token}", srv.Restrict(a, nil, tok.Token(tok.Destroy))).Methods("DELETE")
	auth.HandleFunc("/tokens/revoke", srv.Restrict(a, nil, tok.Token(tok.Destroy))).Methods("DELETE")
	auth.HandleFunc("/apps", srv.Restrict(a, nil, app.Index)).Methods("GET")
	auth.HandleFunc("/apps/create", srv.Restrict(a, nil, app.Create)).Methods("GET")
	auth.HandleFunc("/apps", srv.Restrict(a, nil, app.Store)).Methods("POST")
	auth.HandleFunc("/apps/{app}", srv.Restrict(a, nil, app.App(app.Show))).Methods("GET")
	auth.HandleFunc("/apps/{app}", srv.Restrict(a, nil, app.App(app.Update))).Methods("PATCH")
	auth.HandleFunc("/apps/{app}/revoke", srv.Restrict(a, nil, app.App(app.Update))).Methods("PATCH")
	auth.HandleFunc("/apps/{app}/reset", srv.Restrict(a, nil, app.App(app.Update))).Methods("PATCH")
	auth.HandleFunc("/connections", srv.Restrict(a, nil, tok.Index)).Methods("GET")
	auth.HandleFunc("/connections/{id}", srv.Restrict(a, nil, tok.Token(tok.Show))).Methods("GET")
	auth.HandleFunc("/connections/{id}", srv.Restrict(a, nil, tok.Token(tok.Destroy))).Methods("DELETE")
	auth.Use(srv.CSRF)
}
