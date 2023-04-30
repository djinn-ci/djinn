package http

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/alert"
	"djinn-ci.com/auth"
	"djinn-ci.com/auth/oauth2"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/mail"
	"djinn-ci.com/provider"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/template/form"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"
)

type UI struct {
	*Handler
}

func (h UI) Home(u *auth.User, w http.ResponseWriter, r *http.Request) {
	if u.ID == 0 {
		h.Redirect(w, r, "/login")
		return
	}
	h.Redirect(w, r, "/builds")
}

func (h UI) Register(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if r.Method == "GET" {
		tmpl := template.Register{
			Form:  form.New(sess, r),
			Alert: alert.First(sess),
		}
		h.Template(w, r, &tmpl, http.StatusOK)
		return
	}

	f := RegisterForm{
		DB: h.DB,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		h.FormError(w, r, f, errors.Wrap(err, "Failed to create account"))
		return
	}

	ctx := r.Context()

	u, err := h.Users.Create(ctx, &user.Params{
		Email:    f.Email,
		Username: f.Username,
		Password: f.Password,
	})

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to create account"))
		return
	}

	for _, name := range h.Server.Providers.Names() {
		err := h.Providers.Create(ctx, &provider.Provider{
			UserID:      u.ID,
			Name:        name,
			MainAccount: true,
		})

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to create account"))
			return
		}
	}

	h.Queues.Produce(ctx, "email", user.VerifyMail(h.SMTP.From, webutil.BaseAddress(r), u))

	alert.Flash(sess, alert.Warn, "A verification link has been sent to your email")
	h.Redirect(w, r, "/login")
}

func (h UI) Login(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if r.Method == "GET" {
		names := h.Server.Providers.Names()

		auths := make([]template.AuthForm, 0, len(names)+1)
		auths = append(auths, template.AuthForm{
			Form:     form.New(sess, r),
			Mech:     user.InternalProvider,
			Provider: user.InternalProvider,
		})

		for _, name := range names {
			auths = append(auths, template.AuthForm{
				Form:     form.New(sess, r),
				Mech:     "oauth2." + name,
				Provider: name,
			})
		}

		tmpl := template.Login{
			Alert:       alert.First(sess),
			RedirectURI: r.URL.Query().Get("redirect_uri"),
			Auths:       auths,
		}
		h.Template(w, r, &tmpl, http.StatusOK)
		return
	}

	var f LoginForm

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		h.FormError(w, r, f, errors.Wrap(err, "Failed to authenticate"))
		return
	}

	a, err := h.Auths.Get(f.AuthMech)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to authenticate request"))
		return
	}

	if cli, ok := a.(*oauth2.Authenticator); ok {
		url := cli.AuthURL()

		h.Log.Debug.Println(r.Method, r.URL, "authenticating via oauth2 provider", f.AuthMech)
		h.Log.Debug.Println(r.Method, r.URL, "auth_url =", url)

		http.Redirect(w, r, url, http.StatusSeeOther)
		return
	}

	u, err := a.Auth(r)

	if err != nil {
		if !errors.Is(err, auth.ErrAuth) {
			h.Error(w, r, errors.Wrap(err, "Failed to authenticate request"))
			return
		}
		h.Error(w, r, errors.Benign("Invalid credentials"))
		return
	}

	cookie, err := user.Cookie(u, h.SecureCookie)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to authenticate request"))
		return
	}

	uri := "/builds"

	if f.RedirectURI != "" {
		uri = f.RedirectURI
	}

	h.Log.Debug.Println(r.URL, r.Method, "authenticated as", u.Email, "redirecting to", uri)

	http.SetCookie(w, cookie)
	h.Redirect(w, r, uri)
}

var resetMail = `A request was made to reset your password. Follow the link below to reset your
account's password:

    %s/new_password?token=%s

if you did not make this request then ignore this email.`

func (h UI) PasswordReset(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if r.Method == "GET" {
		tmpl := template.PasswordReset{
			Form:  form.New(sess, r),
			Token: r.URL.Query().Get("token"),
			Alert: alert.First(sess),
		}
		h.Template(w, r, &tmpl, http.StatusOK)
		return
	}

	var f PasswordResetForm

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		h.FormError(w, r, f, errors.Wrap(err, "Failed to reset password"))
		return
	}

	ctx := r.Context()

	u, ok, err := h.Users.Get(ctx, user.WhereEmail(f.Email))

	if err != nil {
		h.FormError(w, r, f, errors.Wrap(err, "Failed to reset password"))
		return
	}

	// Respond with a faux success message - don't want people to try and
	// reverse engineer whether an email exists within the system.
	if !ok {
		alert.Flash(sess, alert.Success, "Password reset instructions sent")
		h.RedirectBack(w, r)
		return
	}

	tok, err := h.Users.ResetPassword(ctx, u)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to reset password"))
		return
	}

	h.Queues.Produce(ctx, "email", &mail.Mail{
		From:    h.SMTP.From,
		To:      []string{u.Email},
		Subject: "Djinn CI - Password reset request",
		Body:    fmt.Sprintf(resetMail, webutil.BaseAddress(r), tok),
	})

	alert.Flash(sess, alert.Success, "Password reset instructions sent")
	h.RedirectBack(w, r)
}

func (h UI) NewPassword(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if r.Method == "GET" {
		tmpl := template.PasswordReset{
			Form:  form.New(sess, r),
			Alert: alert.First(sess),
		}
		h.Template(w, r, &tmpl, http.StatusOK)
		return
	}

	var f NewPasswordForm

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		h.FormError(w, r, f, errors.Wrap(err, "Failed to set new password"))
		return
	}

	if err := h.Users.UpdatePassword(r.Context(), f.Token, f.Password); err != nil {
		cause := errors.Cause(err)

		if errors.Is(cause, user.ErrTokenExpired) {
			alert.Flash(sess, alert.Danger, "Token expired")
			h.RedirectBack(w, r)
			return
		}
		h.Error(w, r, errors.Wrap(err, "Failed to set new password"))
		return
	}

	alert.Flash(sess, alert.Success, "Password updated")
	h.Redirect(w, r, "/login")
}

func (h UI) Settings(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	pp0, err := h.Providers.All(
		r.Context(),
		query.Where("user_id", "=", query.Arg(u.ID)),
		query.OrderAsc("name"),
	)

	if err != nil {
		h.Error(w, r, err)
		return
	}

	providertab := make(map[string]*provider.Provider)

	for _, p := range pp0 {
		providertab[p.Name] = p
	}

	pp := make([]*provider.Provider, 0, len(pp0))

	for _, name := range h.Server.Providers.Names() {
		a, _ := h.Auths.Get("oauth2." + name)
		cli, _ := a.(*oauth2.Authenticator)

		p, ok := providertab[name]

		if !ok {
			pp = append(pp, &provider.Provider{
				Name:    name,
				AuthURL: cli.AuthURL(),
			})
			continue
		}

		p.AuthURL = cli.AuthURL()
		pp = append(pp, p)
	}

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.Settings{
		Page:      tmpl.Page,
		Form:      form.New(sess, r),
		Providers: pp,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Verify(u *auth.User, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, _ := h.Session(r)

	if r.Method == "GET" {
		if err := h.Users.Verify(ctx, r.URL.Query().Get("token")); err != nil {
			cause := errors.Cause(err)
			msg := "Invalid verification token"

			if !errors.Is(cause, database.ErrNoRows) {
				msg = "Failed to verify account"
			}

			h.Error(w, r, errors.Wrap(err, msg))
			return
		}

		alert.Flash(sess, alert.Success, "Account has been verified")
		h.Redirect(w, r, "/settings")
		return
	}

	tok, err := h.Users.RequestVerify(ctx, u.ID)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to send verification email"))
		return
	}

	u.RawData["account_token"] = tok

	h.Queues.Produce(ctx, "email", user.VerifyMail(h.SMTP.From, webutil.BaseAddress(r), u))

	alert.Flash(sess, alert.Success, "Verification email sent to: "+u.Email)
	h.RedirectBack(w, r)
}

func parseSize(s string) (int64, error) {
	siztab := map[string]int64{
		"B":  1,
		"KB": 1 << 10,
		"MB": 1 << 20,
		"GB": 1 << 30,
	}

	pos := -1

	for i, r := range s {
		if r == ' ' {
			continue
		}

		if '0' <= r && r <= '9' {
			continue
		}

		pos = i
		break
	}

	mult := int64(1)

	if pos > -1 {
		var ok bool
		mult, ok = siztab[s[pos:]]

		if !ok {
			return 0, errors.New("Invalid size")
		}
		s = s[:pos]
	}

	i, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)

	if err != nil {
		return 0, err
	}
	return i * mult, nil
}

func (h UI) Cleanup(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := r.ParseForm(); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to save changes"))
		return
	}

	cleanup, err := parseSize(r.PostForm.Get("cleanup"))

	if err != nil {
		errs := make(webutil.ValidationErrors)
		errs.Add("cleanup", err)

		sess.AddFlash(errs, "form_errors")

		h.RedirectBack(w, r)
		return
	}

	u.RawData["cleanup"] = cleanup

	if err := h.Users.Update(r.Context(), u); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to save changes"))
		return
	}

	alert.Flash(sess, alert.Success, "Changes have been saved")
	h.RedirectBack(w, r)
}

func (h UI) Email(u *auth.User, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, _ := h.Session(r)

	if r.Method == "GET" {
		if u.Email != "" {
			h.Redirect(w, r, "/settings")
			return
		}

		tok, err := h.Users.ResetEmail(ctx, u)

		if err != nil {
			h.Error(w, r, err)
			return
		}

		tmpl := template.SetEmail{
			Form:        form.New(sess, r),
			User:        u,
			Alert:       alert.First(sess),
			Token:       tok,
			RedirectURI: r.URL.Query().Get("redirect_uri"),
		}

		h.Template(w, r, &tmpl, http.StatusOK)
		return
	}

	parts := strings.Split(r.Header.Get("Referer"), "/")

	// User email not set during OAuth flow, so rewrite the Referer to be the
	// "/settings" endpoint, this avoids screwing with the OAuth flow.
	if parts[0] == "oauth" {
		r.Header.Set("Referer", "/settings")
	}

	f := EmailForm{
		DB: h.DB,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		if errs, ok := err.(webutil.ValidationErrors); ok {
			webutil.FlashFormWithErrors(sess, f, errs)
			h.RedirectBack(w, r)
			return
		}

		h.Error(w, r, errors.Wrap(err, "Failed to update email"))
		return
	}

	u.Email = f.Email

	if f.Token != "" {
		if err := h.Users.UpdateEmail(ctx, f.Token, f.Email); err != nil {
			if errors.Is(errors.Cause(err), user.ErrTokenExpired) {
				alert.Flash(sess, alert.Danger, "Token expired")
				h.RedirectBack(w, r)
				return
			}

			h.Error(w, r, errors.Wrap(err, "Failed to update email"))
			return
		}
		goto resp
	}

	if err := h.Users.Update(ctx, u); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to update email"))
		return
	}

resp:
	uri := r.Header.Get("Referer")

	if f.RedirectURI != "" {
		uri = f.RedirectURI
	}

	alert.Flash(sess, alert.Success, "Email has been updated")
	h.Redirect(w, r, uri)
}

func (h UI) Password(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	f := PasswordForm{
		User: u,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to update password"))
		return
	}

	ctx := r.Context()

	tok, err := h.Users.ResetPassword(ctx, u)

	if err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to update password"))
		return
	}

	if err := h.Users.UpdatePassword(ctx, tok, f.NewPassword); err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to update password"))
		return
	}

	alert.Flash(sess, alert.Success, "Password has been updated")
	h.RedirectBack(w, r)
}

func (h UI) Delete(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	var f DeleteForm

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to delete account"))
		return
	}

	ctx := r.Context()

	pp, err := h.Providers.All(ctx, query.Where("user_id", "=", query.Arg(u.ID)))

	if err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to delete account"))
		return
	}

	if err := h.Providers.Delete(ctx, pp...); err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to delete account"))
		return
	}

	if err := h.Users.Delete(ctx, u); err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to delete account"))
		return
	}

	h.Queues.Produce(ctx, "email", &mail.Mail{
		From:    h.SMTP.From,
		To:      []string{u.Email},
		Subject: "Djinn CI - Account deleted",
		Body:    "Your Djinn CI account has been deleted, you will no longer be able to access your builds.",
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "user",
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Unix(0, 0),
	})
	alert.Flash(sess, alert.Success, "Account deleted")
	h.Redirect(w, r, "/")
}

func (h UI) Logout(u *auth.User, w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "user",
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Unix(0, 0),
	})
	h.Redirect(w, r, "/login")
}

func RegisterUI(a auth.Authenticator, srv *server.Server) {
	ui := UI{
		Handler: NewHandler(srv),
	}

	srv.Router.HandleFunc("/", srv.Optional(a, ui.Home)).Methods("GET")

	guest := srv.Router.PathPrefix("/").Subrouter()
	guest.HandleFunc("/register", ui.Register).Methods("GET", "POST")
	guest.HandleFunc("/login", ui.Login).Methods("GET", "POST")
	guest.HandleFunc("/password_reset", ui.PasswordReset).Methods("GET", "POST")
	guest.HandleFunc("/new_password", ui.NewPassword).Methods("GET", "POST")
	guest.Use(srv.Guest(a), srv.CSRF)

	sr := srv.Router.PathPrefix("/").Subrouter()
	sr.HandleFunc("/settings", srv.Restrict(a, nil, ui.Settings)).Methods("GET")
	sr.HandleFunc("/settings/verify", srv.Restrict(a, nil, ui.Verify)).Methods("GET", "POST")
	sr.HandleFunc("/settings/cleanup", srv.Restrict(a, nil, ui.Cleanup)).Methods("PATCH")
	sr.HandleFunc("/settings/email", srv.Restrict(a, nil, ui.Email)).Methods("GET")
	sr.HandleFunc("/settings/email", srv.Restrict(a, nil, ui.Email)).Methods("PATCH")
	sr.HandleFunc("/settings/password", srv.Restrict(a, nil, ui.Password)).Methods("PATCH")
	sr.HandleFunc("/settings/delete", srv.Restrict(a, nil, ui.Delete)).Methods("DELETE")
	sr.HandleFunc("/logout", srv.Restrict(a, nil, ui.Logout)).Methods("POST")
	sr.Use(srv.CSRF)
}
