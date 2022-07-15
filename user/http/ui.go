package http

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/alert"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/mail"
	"djinn-ci.com/provider"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/user"
	usertemplate "djinn-ci.com/user/template"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type UI struct {
	*Handler
}

func (h UI) Home(u *user.User, w http.ResponseWriter, r *http.Request) {
	if u.ID == 0 {
		h.Redirect(w, r, "/login")
		return
	}
	h.Redirect(w, r, "/builds")
}

var verifyMail = `To secure your account please verify your email. Click the link below to
verify your account's email address,

    %s/settings/verify?token=%s`

// Register replies to both GET and POST requests. On GET requests, the page for
// account registration is sent. On POST requests an account will be created,
// and a redirect to /login will be sent in the response.
func (h UI) Register(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		p := &usertemplate.Register{
			Form: template.Form{
				CSRF:   csrf.TemplateField(r),
				Errors: webutil.FormErrors(sess),
				Fields: webutil.FormFields(sess),
			},
			Alert: alert.First(sess),
		}
		save(r, w)
		webutil.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	var f RegisterForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to create account")
		h.RedirectBack(w, r)
		return
	}

	v := RegisterValidator{
		Users: h.Users,
		Form:  f,
	}

	if err := webutil.Validate(v); err != nil {
		verrs := err.(webutil.ValidationErrors)

		if errs, ok := verrs["fatal"]; ok {
			h.Log.Error.Println(r.Method, r.URL, errors.Slice(errs))
			alert.Flash(sess, alert.Danger, "Failed to create account")
			h.RedirectBack(w, r)
			return
		}

		webutil.FlashFormWithErrors(sess, f, verrs)
		h.RedirectBack(w, r)
		return
	}

	u, tok, err := h.Users.Create(user.Params{
		Email:    f.Email,
		Username: f.Username,
		Password: f.Password,
	})

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to create account")
		h.RedirectBack(w, r)
		return
	}

	for name := range h.Server.Providers.All() {
		params := provider.Params{
			UserID:      u.ID,
			Name:        name,
			MainAccount: true,
		}

		if _, err := h.Providers.Create(params); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to create account")
			return
		}
	}

	h.Queues.Produce(r.Context(), "email", &mail.Mail{
		From:    h.SMTP.From,
		To:      []string{u.Email},
		Subject: "Djinn CI - Verify email",
		Body:    fmt.Sprintf(verifyMail, webutil.BaseAddress(r), tok),
	})

	alert.Flash(sess, alert.Warn, "A verification link has been sent to your email, use this to verify your account")
	h.Redirect(w, r, "/login")
}

// Login replies to both GET and POST request. On GET requests, the page for
// logging in is sent. On POST requests an login attempt will be made, on
// success, a redirect to /builds will be sent in the response.
func (h UI) Login(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		if provider := mux.Vars(r)["provider"]; provider != "" {
			h.Log.Debug.Println(r.Method, r.URL, "provider =", provider)

			cli, err := h.Server.Providers.Get(provider)

			if err != nil {
				alert.Flash(sess, alert.Warn, "Unknown provider: " + provider)
				h.Redirect(w, r, "/login")
				return
			}

			url := cli.AuthURL()

			h.Log.Debug.Println(r.Method, r.URL, "auth_url =", url)

			http.Redirect(w, r, url, http.StatusSeeOther)
			return
		}

		names := h.Server.Providers.Names()

		pp := make([]*provider.Provider, 0, len(names))

		for _, name := range names {
			pp = append(pp, &provider.Provider{
				Name:    name,
			})
		}

		p := &usertemplate.Login{
			Form: template.Form{
				CSRF:   csrf.TemplateField(r),
				Errors: webutil.FormErrors(sess),
				Fields: webutil.FormFields(sess),
			},
			Alert:       alert.First(sess),
			RedirectURI: r.URL.Query().Get("redirect_uri"),
			Providers:   pp,
		}
		save(r, w)
		webutil.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	var f LoginForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Unexpected error occurred during authentication")
		h.RedirectBack(w, r)
		return
	}

	if err := webutil.Validate(f); err != nil {
		verrs := err.(webutil.ValidationErrors)
		webutil.FlashFormWithErrors(sess, f, verrs)
		h.RedirectBack(w, r)
		return
	}

	u, err := h.Users.Auth(f.Handle, f.Password)

	if err != nil {
		if !errors.Is(err, user.ErrAuth) {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Unexpected error occurred during authentication")
			h.RedirectBack(w, r)
			return
		}

		errs := webutil.NewValidationErrors()
		errs.Add("handle", user.ErrAuth)
		errs.Add("password", user.ErrAuth)

		webutil.FlashFormWithErrors(sess, f, errs)
		h.RedirectBack(w, r)
		return
	}

	id := strconv.FormatInt(u.ID, 10)

	encoded, err := h.SecureCookie.Encode("user", id)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Unexpected error occurred during authentication")
		h.RedirectBack(w, r)
		return
	}

	uri := "/builds"

	if f.RedirectURI != "" {
		uri = f.RedirectURI
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "user",
		HttpOnly: true,
		MaxAge:   user.MaxAge,
		Expires:  time.Now().Add(time.Duration(user.MaxAge) * time.Second),
		Value:    encoded,
	})
	h.Redirect(w, r, uri)
}

var resetMail = `A request was made to reset your password. Follow the link below to reset your
account's password:

    %s/new_password?token=%s

if you did not make this request then ignore this email.`

func (h UI) PasswordReset(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		p := &usertemplate.PasswordReset{
			Form: template.Form{
				CSRF:   csrf.TemplateField(r),
				Errors: webutil.FormErrors(sess),
				Fields: webutil.FormFields(sess),
			},
			Alert: alert.First(sess),
		}
		save(r, w)
		webutil.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	var f PasswordResetForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to reset password")
		h.RedirectBack(w, r)
		return
	}

	if err := webutil.Validate(f); err != nil {
		verrs := err.(webutil.ValidationErrors)
		webutil.FlashFormWithErrors(sess, f, verrs)
		h.RedirectBack(w, r)
		return
	}

	u, ok, err := h.Users.Get(user.WhereEmail(f.Email))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to reset password")
		h.RedirectBack(w, r)
		return
	}

	// Respond with a faux success message - don't want people to try and
	// reverse engineer whether an email exists within the system.
	if !ok {
		alert.Flash(sess, alert.Success, "Password reset instructions sent")
		h.RedirectBack(w, r)
		return
	}

	tok, err := h.Users.ResetPassword(u.ID)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to reset password")
		h.RedirectBack(w, r)
		return
	}

	m := mail.Mail{
		From:    h.SMTP.From,
		To:      []string{u.Email},
		Subject: "Djinn CI - Password reset request",
		Body:    fmt.Sprintf(resetMail, webutil.BaseAddress(r), tok),
	}

	if err := m.Send(h.SMTP.Client); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to reset password")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Password reset instructions sent")
	h.RedirectBack(w, r)
}

func (h UI) NewPassword(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		p := &usertemplate.NewPassword{
			Form: template.Form{
				CSRF:   csrf.TemplateField(r),
				Errors: webutil.FormErrors(sess),
				Fields: webutil.FormFields(sess),
			},
			Token: r.URL.Query().Get("token"),
			Alert: alert.First(sess),
		}
		save(r, w)
		webutil.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	var f NewPasswordForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to set new password")
		return
	}

	if err := webutil.Validate(f); err != nil {
		verrs := err.(webutil.ValidationErrors)
		webutil.FlashFormWithErrors(sess, f, verrs)
		h.RedirectBack(w, r)
		return
	}

	if err := h.Users.UpdatePassword(f.Token, f.Password); err != nil {
		cause := errors.Cause(err)

		if errors.Is(cause, user.ErrTokenExpired) {
			alert.Flash(sess, alert.Danger, "Token expired")
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to set new password")
		return
	}

	alert.Flash(sess, alert.Success, "Password updated")
	h.Redirect(w, r, "/login")
}

func (h UI) getProviders(u *user.User) ([]*provider.Provider, error) {
	pp0, err := h.Providers.All(
		query.Where("user_id", "=", query.Arg(u.ID)),
		query.OrderAsc("name"),
	)

	if err != nil {
		return nil, errors.Err(err)
	}

	m := make(map[string]*provider.Provider)

	for _, p := range pp0 {
		m[p.Name] = p
	}

	names := h.Server.Providers.Names()
	clients := h.Server.Providers.All()

	pp := make([]*provider.Provider, 0, len(names))

	for _, name := range names {
		if _, ok := m[name]; !ok {
			pp = append(pp, &provider.Provider{
				Name:    name,
				AuthURL: clients[name].AuthURL(),
			})
			continue
		}
		p := m[name]
		p.AuthURL = clients[name].AuthURL()

		pp = append(pp, p)
	}
	return pp, nil
}

var (
	sudoTimestamp = "sudo_timestamp"
	sudoToken     = "sudo_token"
	sudoUrl       = "sudo_url"
	sudoReferer   = "sudo_referer"
)

func (h UI) Sudo(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	tok0, ok := sess.Values[sudoToken]

	if !ok {
		h.NotFound(w, r)
		return
	}

	url0, ok := sess.Values[sudoUrl]

	if !ok {
		h.NotFound(w, r)
		return
	}

	ref0, ok := sess.Values[sudoReferer]

	if !ok {
		h.NotFound(w, r)
		return
	}

	delete(sess.Values, sudoToken)

	tok, _ := tok0.(string)
	url, _ := url0.(string)
	ref, _ := ref0.(string)

	if r.Method == "GET" {
		sess.Values[sudoToken] = tok

		p := &usertemplate.Sudo{
			Form: template.Form{
				CSRF:   csrf.TemplateField(r),
				Errors: webutil.FormErrors(sess),
				Fields: webutil.FormFields(sess),
			},
			Alert:       alert.First(sess),
			User:        u,
			SudoURL:     url,
			SudoReferer: ref,
			SudoToken:   tok,
		}

		save(r, w)
		webutil.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	var f SudoForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Unexpected error occurred during authorization")
		sess.Values[sudoToken] = h.generateSudoToken()
		h.Redirect(w, r, "/sudo")
		return
	}

	h.Log.Debug.Println(r.Method, r.URL, "authorizing sudo request")

	v := SudoValidator{
		Form:  f,
		User:  u,
		Token: tok,
	}

	if err := webutil.Validate(v); err != nil {
		verrs := err.(webutil.ValidationErrors)
		webutil.FlashFormWithErrors(sess, f, verrs)
		sess.Values[sudoToken] = h.generateSudoToken()
		h.Redirect(w, r, "/sudo")
		return
	}

	expires := time.Now().Add(time.Minute * 30)

	h.Log.Debug.Println(r.Method, r.URL, "sudo request authorized, expires at", expires)

	sess.Values[sudoTimestamp] = expires
	h.Redirect(w, r, f.URL)
}

func (h UI) Settings(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	pp, err := h.getProviders(u)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	csrf := csrf.TemplateField(r)

	p := &usertemplate.Settings{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
		Providers: pp,
	}

	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Verify(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if r.Method == "GET" {
		if err := h.Users.Verify(r.URL.Query().Get("token")); err != nil {
			cause := errors.Cause(err)

			msg := "Invalid verification token"

			if !errors.Is(cause, database.ErrNotFound) {
				h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
				msg = "Failed to verify account"
			}

			alert.Flash(sess, alert.Danger, msg)
			h.Redirect(w, r, "/settings")
			return
		}

		alert.Flash(sess, alert.Success, "Account has been verified")
		h.Redirect(w, r, "/settings")
		return
	}

	tok, err := h.Users.RequestVerify(u.ID)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to send verification email")
		h.RedirectBack(w, r)
		return
	}

	m := mail.Mail{
		From:    h.SMTP.From,
		To:      []string{u.Email},
		Subject: "Djinn CI - Verify email",
		Body:    fmt.Sprintf(verifyMail, webutil.BaseAddress(r), tok),
	}

	if err := m.Send(h.SMTP.Client); err != nil {
		cause := errors.Cause(err)

		if rcpterrs, ok := cause.(*mail.ErrRcpts); ok {
			h.Log.Error.Println(r.Method, r.URL, "Failed to send verification email to "+rcpterrs.Error())
			goto resp
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to send verification email")
		h.RedirectBack(w, r)
		return
	}

resp:
	alert.Flash(sess, alert.Success, "Verification email sent to: "+u.Email)
	h.RedirectBack(w, r)
}

func parseSize(s string) (int64, error) {
	siztab := map[string]int64{
		"B":  1,
		"KB": 1<<10,
		"MB": 1<<20,
		"GB": 1<<30,
	}

	pos := -1

	for i, r := range s {
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
			return 0, errors.New("invalid size")
		}
		s = s[:pos]
	}

	i, err := strconv.ParseInt(s, 10, 64)

	if err != nil {
		return 0, err
	}
	return i*mult, nil
}

func (h UI) Cleanup(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := r.ParseForm(); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to save changes")
		h.RedirectBack(w, r)
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

	params := user.Params{
		Email:   u.Email,
		Cleanup: cleanup,
	}

	if err := h.Users.Update(u.ID, params); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to save changes")
		h.Redirect(w, r, "/settings")
		return
	}

	alert.Flash(sess, alert.Success, "Changes have been saved")
	h.RedirectBack(w, r)
}

func (h UI) Email(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		if u.Email != "" {
			h.Redirect(w, r, "/settings")
			return
		}

		tok, err := h.Users.ResetEmail(u.ID)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		p := &usertemplate.Email{
			Form: template.Form{
				CSRF:   csrf.TemplateField(r),
				Errors: webutil.FormErrors(sess),
				Fields: webutil.FormFields(sess),
			},
			User:        u,
			Token:       tok,
			RedirectURI: r.URL.Query().Get("redirect_uri"),
		}
		save(r, w)
		webutil.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	parts := strings.Split(r.Header.Get("Referer"), "/")

	// User email not set during OAuth flow, so rewrite the Referer to be the
	// "/settings" endpoint, this avoids screwing with the OAuth flow.
	if parts[0] == "oauth" {
		r.Header.Set("Referer", "/settings")
	}

	var f EmailForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Success, "Failed to update email")
		h.RedirectBack(w, r)
		return
	}

	v := EmailValidator{
		Users: h.Users,
		User:  u,
		Form:  f,
	}

	if err := webutil.Validate(v); err != nil {
		verrs := err.(webutil.ValidationErrors)
		webutil.FlashFormWithErrors(sess, f, verrs)
		h.RedirectBack(w, r)
		return
	}

	params := user.Params{
		Email:   f.Email,
		Cleanup: u.Cleanup,
	}

	if f.Token != "" {
		if err := h.Users.UpdateEmail(f.Token, f.Email); err != nil {
			if errors.Is(errors.Cause(err), user.ErrTokenExpired) {
				alert.Flash(sess, alert.Danger, "Token expired")
				h.RedirectBack(w, r)
				return
			}

			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to update email")
			h.RedirectBack(w, r)
			return
		}
		goto resp
	}

	if err := h.Users.Update(u.ID, params); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to update email")
		h.RedirectBack(w, r)
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

func (h UI) Password(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	var f PasswordForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to update password")
		h.RedirectBack(w, r)
		return
	}

	v := PasswordValidator{
		User:  u,
		Users: h.Users,
		Form:  f,
	}

	if err := webutil.Validate(v); err != nil {
		verrs := err.(webutil.ValidationErrors)
		webutil.FlashFormWithErrors(sess, f, verrs)
		h.RedirectBack(w, r)
		return
	}

	params := user.Params{
		Email:    u.Email,
		Cleanup:  u.Cleanup,
		Password: f.NewPassword,
	}

	if err := h.Users.Update(u.ID, params); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to update password")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Password has been updated")
	h.RedirectBack(w, r)
}

func (h UI) Delete(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	var f DeleteForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Success, "Failed to delete account")
		h.RedirectBack(w, r)
		return
	}

	v := DeleteValidator{
		User: u,
		Form: f,
	}

	if err := webutil.Validate(v); err != nil {
		verrs := err.(webutil.ValidationErrors)
		webutil.FlashFormWithErrors(sess, f, verrs)
		h.RedirectBack(w, r)
		return
	}

	pp, err := h.Providers.All(query.Where("user_id", "=", query.Arg(u.ID)))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to delete account")
		h.RedirectBack(w, r)
		return
	}

	ids := make([]int64, 0, len(pp))

	for _, p := range pp {
		ids = append(ids, p.ID)
	}

	if err := h.Providers.DeleteAll(ids...); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to delete account")
		h.RedirectBack(w, r)
		return
	}

	if err := h.Users.Delete(u.ID); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to delete account")
		h.RedirectBack(w, r)
		return
	}

	m := mail.Mail{
		From:    h.SMTP.From,
		To:      []string{u.Email},
		Subject: "Djinn CI - Account deleted",
		Body:    "Your Djinn CI account has been deleted, you will no longer be able to access your builds.",
	}

	if err := m.Send(h.SMTP.Client); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to delete account")
		h.RedirectBack(w, r)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "user",
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Unix(0, 0),
	})
	alert.Flash(sess, alert.Success, "Account deleted")
	h.Redirect(w, r, "/")
}

func (h UI) Logout(u *user.User, w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "user",
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Unix(0, 0),
	})
	h.Redirect(w, r, "/login")
}

func RegisterUI(srv *server.Server) {
	ui := UI{
		Handler: NewHandler(srv),
	}

	srv.Router.HandleFunc("/", ui.WithOptionalUser(ui.Home)).Methods("GET")

	guest := srv.Router.PathPrefix("/").Subrouter()
	guest.HandleFunc("/register", ui.Register).Methods("GET", "POST")
	guest.HandleFunc("/login", ui.Login).Methods("GET", "POST")
	guest.HandleFunc("/login/{provider}", ui.Login).Methods("GET")
	guest.HandleFunc("/password_reset", ui.PasswordReset).Methods("GET", "POST")
	guest.HandleFunc("/new_password", ui.NewPassword).Methods("GET", "POST")
	guest.Use(ui.Guest, srv.CSRF)

	auth := srv.Router.PathPrefix("/").Subrouter()
	auth.HandleFunc("/sudo", ui.WithUser(ui.Sudo)).Methods("GET", "POST")
	auth.HandleFunc("/settings", ui.WithUser(ui.Settings)).Methods("GET")
	auth.HandleFunc("/settings/verify", ui.WithUser(ui.Verify)).Methods("GET", "POST")
	auth.HandleFunc("/settings/cleanup", ui.WithUser(ui.Cleanup)).Methods("PATCH")
	auth.HandleFunc("/settings/email", ui.WithUser(ui.Email)).Methods("GET")
	auth.HandleFunc("/settings/email", ui.WithUser(ui.Email)).Methods("PATCH")
	auth.HandleFunc("/settings/password", ui.WithUser(ui.Password)).Methods("PATCH")
	auth.HandleFunc("/settings/delete", ui.WithUser(ui.Delete)).Methods("POST")
	auth.HandleFunc("/logout", ui.WithUser(ui.Logout)).Methods("POST")
	auth.Use(srv.CSRF)
}
