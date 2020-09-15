package handler

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/mail"
	"github.com/andrewpillar/djinn/provider"
	"github.com/andrewpillar/djinn/template"
	"github.com/andrewpillar/djinn/user"
	usertemplate "github.com/andrewpillar/djinn/user/template"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
)

var (
	resetMail = `A request was made to reset your password. Follow the link below to reset your
account's password:

    %s/new_password?token=%s

if you did not make this request then ignore this email.`

	verifyMail = `To secure your account please verify your email. Click the link below to
verify your account's email address,

    %s/settings/verify?token=%s`
)

type User struct {
	web.Handler

	Prefix   string
	Registry *provider.Registry
}

func (h User) Register(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		p := &usertemplate.Register{
			Form: template.Form{
				CSRF:   string(csrf.TemplateField(r)),
				Errors: web.FormErrors(sess),
				Fields: web.FormFields(sess),
			},
			Alert: web.Alert(sess),
		}
		save(r, w)
		web.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	f := &user.RegisterForm{Users: h.Users}

	if err := form.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create account"), "alert")
		h.RedirectBack(w, r)
		return
	}

	u, tok, err := h.Users.Create(f.Email, f.Username, []byte(f.Password))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create account"), "alert")
		h.RedirectBack(w, r)
		return
	}

	m := mail.Mail{
		From:    h.SMTP.From,
		To:      []string{u.Email},
		Subject: "Djinn - Verify email",
		Body:    fmt.Sprintf(verifyMail, web.BaseAddress(r), hex.EncodeToString(tok)),
	}

	if err := m.Send(h.SMTP.Client); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create account"), "alert")
		h.RedirectBack(w, r)
		return
	}

	providers := provider.NewStore(h.DB, u)

	for name := range h.Registry.All() {
		if _, err := providers.Create(0, name, nil, nil, false, false); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to create account"), "alert")
			return
		}
	}

	sess.AddFlash(template.Warn("A verification link has been sent to your email, use this to verify your account"))
	h.Redirect(w, r, "/login")
}

func (h User) Login(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		clis := h.Registry.All()

		order := make([]string, 0, len(clis))

		for name := range clis {
			order = append(order, name)
		}

		sort.Strings(order)

		pp := make([]*provider.Provider, 0, len(clis))

		for _, name := range order {
			cli := clis[name]

			pp = append(pp, &provider.Provider{
				Name:    name,
				AuthURL: cli.AuthURL(),
			})
		}

		p := &usertemplate.Login{
			Form: template.Form{
				CSRF:   string(csrf.TemplateField(r)),
				Errors: web.FormErrors(sess),
				Fields: web.FormFields(sess),
			},
			Alert:     web.Alert(sess),
			Providers: pp,
		}
		save(r, w)
		web.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	f := &user.LoginForm{}

	if err := form.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Unexpected error occurred during authentication"), "alert")
		h.RedirectBack(w, r)
		return
	}

	u, err := h.Users.Auth(f.Handle, f.Password)

	if err != nil {
		cause := errors.Cause(err)

		if cause != user.ErrAuth {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Unexpected error occurred during authentication"), "alert")
			h.RedirectBack(w, r)
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

	id := strconv.FormatInt(u.ID, 10)

	encoded, err := h.SecureCookie.Encode("user", id)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Unexpected error occurred during authentication"), "alert")
		h.RedirectBack(w, r)
		return
	}

	uri := "/"

	if uri1 := r.URL.Query().Get("redirect_uri"); uri1 != "" {
		uri = uri1
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

func (h User) NewPassword(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		p := &usertemplate.NewPassword{
			Form: template.Form{
				CSRF:   string(csrf.TemplateField(r)),
				Errors: web.FormErrors(sess),
				Fields: web.FormFields(sess),
			},
			Token: r.URL.Query().Get("token"),
			Alert: web.Alert(sess),
		}
		save(r, w)
		web.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	f := &user.NewPasswordForm{}

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

	tok, err := hex.DecodeString(f.Token)

	if err != nil {
		sess.AddFlash(template.Danger("Invalid token"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := h.Users.UpdatePassword(tok, []byte(f.Password)); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Password updated"), "alert")
	h.Redirect(w, r, "/login")
}

func (h User) PasswordReset(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		p := &usertemplate.PasswordReset{
			Form: template.Form{
				CSRF:   string(csrf.TemplateField(r)),
				Errors: web.FormErrors(sess),
				Fields: web.FormFields(sess),
			},
			Alert: web.Alert(sess),
		}
		save(r, w)
		web.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	f := &user.PasswordResetForm{}

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

	u, err := h.Users.Get(query.Where("email", "=", f.Email))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		h.RedirectBack(w, r)
		return
	}

	if u.IsZero() {
		sess.AddFlash(template.Success("Password reset instructions sent"), "alert")
		h.RedirectBack(w, r)
		return
	}

	tok, err := h.Users.ResetPassword(u.ID)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		h.RedirectBack(w, r)
		return
	}

	m := mail.Mail{
		From:    h.SMTP.From,
		To:      []string{u.Email},
		Subject: "Djinn - Password reset request",
		Body:    fmt.Sprintf(resetMail, web.BaseAddress(r), hex.EncodeToString(tok)),
	}

	if err := m.Send(h.SMTP.Client); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		h.RedirectBack(w, r)
		return
	}
	sess.AddFlash(template.Success("Password reset instructions sent"), "alert")
	h.RedirectBack(w, r)
}

func (h User) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "user",
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Unix(0, 0),
	})
	h.Redirect(w, r, "/")
}

func (h User) Settings(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	m := make(map[string]*provider.Provider)

	pp, err := provider.NewStore(h.DB, u).All(query.OrderAsc("name"))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	for _, p := range pp {
		m[p.Name] = p
	}

	clis := h.Registry.All()

	order := make([]string, 0, len(clis))

	for name, cli := range clis {
		order = append(order, name)

		if _, ok := m[name]; !ok {
			m[name] = &provider.Provider{
				Name:    name,
				AuthURL: cli.AuthURL(),
			}
		} else {
			m[name].AuthURL = cli.AuthURL()
		}
	}

	sort.Strings(order)

	p := &usertemplate.Settings{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: web.FormErrors(sess),
			Fields: web.FormFields(sess),
		},
		Providers: make([]*provider.Provider, 0, len(clis)),
	}

	for _, name := range order {
		p.Providers = append(p.Providers, m[name])
	}

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

// Verify will either send a verification email to the user, or verify the
// user's account. If the tok query parameter is in the current request then
// an attempt will be made to verify the user's account. If the tok query
// parameter is not in the current request then the verification email is
// sent for account verification.
func (h User) Verify(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if r.Method == "POST" {
		u, ok := user.FromContext(r.Context())

		if !ok {
			h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
		}

		tok, err := h.Users.RequestVerify(u.ID)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to send verification email"), "alert")
			h.RedirectBack(w, r)
			return
		}

		m := mail.Mail{
			From:    h.SMTP.From,
			To:      []string{u.Email},
			Subject: "Djinn - Verify email",
			Body:    fmt.Sprintf(verifyMail, web.BaseAddress(r), hex.EncodeToString(tok)),
		}

		if err := m.Send(h.SMTP.Client); err != nil {
			cause := errors.Cause(err)

			if rcpterrs, ok := cause.(*mail.ErrRcpts); ok {
				h.Log.Error.Println(r.Method, r.URL, "Failed to send verification email to " + rcpterrs.Error())
				goto resp
			}

			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to send verification email"), "alert")
			h.RedirectBack(w, r)
			return
		}

resp:
		sess.AddFlash(template.Success("Verification email sent to: " + u.Email), "alert")
		h.RedirectBack(w, r)
		return
	}

	b, err := hex.DecodeString(r.URL.Query().Get("token"))

	if err != nil {
		if errors.Cause(err) == user.ErrTokenExpired {
			sess.AddFlash(template.Danger("Token expired, resend verification email"), "alert")
			h.Redirect(w, r, "/settings")
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to verify account"), "alert")
		h.Redirect(w, r, "/settings")
		return
	}

	if err := h.Users.Verify(b); err != nil {
		cause := errors.Cause(err)

		if cause == database.ErrNotFound {
			h.Redirect(w, r, "/settings")
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to verify account"), "alert")
		h.Redirect(w, r, "/settings")
		return
	}

	sess.AddFlash(template.Success("Account has been verified"), "alert")
	h.Redirect(w, r, "/settings")
}

func (h User) Email(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	f := &user.EmailForm{
		User:  u,
		Users: h.Users,
	}

	if err := form.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update email"), "alert")
		h.RedirectBack(w, r)
		return
	}

	u.Email = f.Email
	u.UpdatedAt = time.Now()

	if err := h.Users.Update(u.ID, f.Email, []byte(f.VerifyPassword)); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update account"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Email has been updated"), "alert")
	h.RedirectBack(w, r)
}

func (h User) Password(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	f := &user.PasswordForm{
		User:  u,
		Users: h.Users,
	}

	if err := form.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update password"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := h.Users.Update(u.ID, u.Email, []byte(f.NewPassword)); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update password"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Password has been updated"), "alert")
	h.RedirectBack(w, r)
}

func (h User) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	f := &user.DeleteForm{}

	if err := form.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to delete account"), "alert")
		h.RedirectBack(w, r)
		return
	}

	providers := provider.NewStore(h.DB, u)

	pp, err := providers.All()

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to delete account"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := providers.Delete(pp...); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to delete account"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := h.Users.Delete(u.ID, []byte(f.Password)); err != nil {
		cause := errors.Cause(err)

		if cause == user.ErrAuth {
			errs := form.NewErrors()
			errs.Put("delete_password", errors.New("Invalid password"))

			web.FlashFormWithErrors(sess, f, errs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to delete account"), "alert")
		h.RedirectBack(w, r)
		return
	}

	m := mail.Mail{
		From:    h.SMTP.From,
		To:      []string{u.Email},
		Subject: "Djinn - Account deleted",
		Body:    "Your Djinn account has been deleted, you will no longer be able to access your builds.",
	}

	if err := m.Send(h.SMTP.Client); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to delete account"), "alert")
		h.RedirectBack(w, r)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "user",
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Unix(0, 0),
	})
	sess.AddFlash(template.Success("Account deleted"), "alert")
	h.Redirect(w, r, "/")
}
