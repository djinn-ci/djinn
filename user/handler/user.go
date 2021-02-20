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
	"github.com/andrewpillar/djinn/mail"
	"github.com/andrewpillar/djinn/provider"
	"github.com/andrewpillar/djinn/template"
	"github.com/andrewpillar/djinn/user"
	usertemplate "github.com/andrewpillar/djinn/user/template"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

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

// User is the handler for handling requests made for managing registration,
// authentication and account management.
type User struct {
	web.Handler

	registry *provider.Registry
}

func New(h web.Handler, registry *provider.Registry) User {
	return User{
		Handler:  h,
		registry: registry,
	}
}

// Home is the handler for the "/" route. This will either redirect to the
// login page, or the builds overview depending on whether or not the user
// is logged in.
func (h User) Home(w http.ResponseWriter, r *http.Request) {
	_, ok, err := h.UserFromRequest(w, r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		h.RedirectBack(w, r)
		return
	}

	if !ok {
		h.Redirect(w, r, "/login")
		return
	}
	h.Redirect(w, r, "/builds")
}

// Register will serve the HTML response for registering an account on a GET
// request. On a POST request it will validate the user's account being created
// and attempt to create a new user. On successful registration an email will be
// sent for the account to be verified, and the user will be redirect to the
// login page.
func (h User) Register(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		p := &usertemplate.Register{
			Form: template.Form{
				CSRF:   csrf.TemplateField(r),
				Errors: webutil.FormErrors(sess),
				Fields: webutil.FormFields(sess),
			},
			Alert: web.Alert(sess),
		}
		save(r, w)
		webutil.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	f := &user.RegisterForm{Users: h.Users}

	if err := webutil.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Message: "Failed to create account",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	u, tok, err := h.Users.Create(f.Email, f.Username, []byte(f.Password))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Message: "Failed to create account",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	m := mail.Mail{
		From:    h.SMTP.From,
		To:      []string{u.Email},
		Subject: "Djinn - Verify email",
		Body:    fmt.Sprintf(verifyMail, webutil.BaseAddress(r), hex.EncodeToString(tok)),
	}

	if err := m.Send(h.SMTP.Client); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Message: "Failed to create account",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	providers := provider.NewStore(h.DB, u)

	for name := range h.registry.All() {
		if _, err := providers.Create(0, name, nil, nil, true, false); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Message: "Failed to create account",
			}, "alert")
			return
		}
	}

	sess.AddFlash(template.Alert{
		Level:   template.Warn,
		Message: "A verification link has been sent to your email, use this to verify your accont",
	}, "alert")
	h.Redirect(w, r, "/login")
}

// Login will serve the HTML response for logging in on a GET request. This
// will either allow for direct authentication against the server itself or
// will allow for logging in via a third party provider. On a POST request it
// will attempt to authenticate the user. On successful authentication the user
// is set in the session, and redirected to the main dashboard.
func (h User) Login(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		clis := h.registry.All()

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
				CSRF:   csrf.TemplateField(r),
				Errors: webutil.FormErrors(sess),
				Fields: webutil.FormFields(sess),
			},
			Alert:       web.Alert(sess),
			RedirectURI: r.URL.Query().Get("redirect_uri"),
			Providers:   pp,
		}
		save(r, w)
		webutil.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	f := &user.LoginForm{}

	if err := webutil.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Message: "Unexpected error occurred during authentication",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	u, err := h.Users.Auth(f.Handle, f.Password)

	if err != nil {
		cause := errors.Cause(err)

		if cause != user.ErrAuth {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Message: "Unexpected error occurred during authentication",
			}, "alert")
			h.RedirectBack(w, r)
			return
		}

		errs := webutil.NewErrors()
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
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Message: "Unexpected error occurred during authentication",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	uri := "/builds"

	if uri1 := f.RedirectURI; uri1 != "" {
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

// NewPassword serves the HTML response for setting a new password on a GET
// request. On a POST request this will attempt to set the new password the
// user specified for their account, if the token given to them via email
// is valid.
func (h User) NewPassword(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		p := &usertemplate.NewPassword{
			Form: template.Form{
				CSRF:   csrf.TemplateField(r),
				Errors: webutil.FormErrors(sess),
				Fields: webutil.FormFields(sess),
			},
			Token: r.URL.Query().Get("token"),
			Alert: web.Alert(sess),
		}
		save(r, w)
		webutil.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	f := &user.NewPasswordForm{}

	if err := webutil.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
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
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Invalid token",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := h.Users.UpdatePassword(tok, []byte(f.Password)); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Password updated",
	}, "alert")
	h.Redirect(w, r, "/login")
}

// PasswordReset will serve the HTML response for requesting a password reset
// token as part of the forgotten password flow. On a POST request this will
// generate a new account token that is used for authenticating the user
// resetting the password when they cannot otherwise provide meaningful
// authentication. The token that is generated expires after a minute.
func (h User) PasswordReset(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		p := &usertemplate.PasswordReset{
			Form: template.Form{
				CSRF:   csrf.TemplateField(r),
				Errors: webutil.FormErrors(sess),
				Fields: webutil.FormFields(sess),
			},
			Alert: web.Alert(sess),
		}
		save(r, w)
		webutil.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	f := &user.PasswordResetForm{}

	if err := webutil.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		h.RedirectBack(w, r)
		return
	}

	u, err := h.Users.Get(query.Where("email", "=", query.Arg(f.Email)))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		h.RedirectBack(w, r)
		return
	}

	if u.IsZero() {
		sess.AddFlash(template.Alert{
			Level:   template.Success,
			Close:   true,
			Message: "Password reset instructions sent",
		}, "alert")
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
		Body:    fmt.Sprintf(resetMail, webutil.BaseAddress(r), hex.EncodeToString(tok)),
	}

	if err := m.Send(h.SMTP.Client); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		h.RedirectBack(w, r)
		return
	}
	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Password reset instructions sent",
	}, "alert")
	h.RedirectBack(w, r)
}

// Logout will log the user out of the current session.
func (h User) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "user",
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Unix(0, 0),
	})
	h.Redirect(w, r, "/login")
}

func (h User) GetProviders(u *user.User) ([]*provider.Provider, error) {
	m := make(map[string]*provider.Provider)

	pp0, err := provider.NewStore(h.DB, u).All(query.OrderAsc("name"))

	if err != nil {
		return nil, errors.Err(err)
	}

	for _, p := range pp0 {
		m[p.Name] = p
	}

	clis := h.registry.All()

	order := make([]string, 0, len(clis))

	for name, cli := range clis {
		order = append(order, name)

		if _, ok := m[name]; !ok {
			m[name] = &provider.Provider{
				Name:    name,
				AuthURL: cli.AuthURL(),
			}
			continue
		}
		m[name].AuthURL = cli.AuthURL()
	}

	sort.Strings(order)

	pp := make([]*provider.Provider, 0, len(clis))

	for _, name := range order {
		pp = append(pp, m[name])
	}
	return pp, nil
}

// Settings will serve the HTML response showing the settings page for managing
// a user's account.
func (h User) Settings(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	pp, err := h.GetProviders(u)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
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

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
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
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to send verification email",
			}, "alert")
			h.RedirectBack(w, r)
			return
		}

		m := mail.Mail{
			From:    h.SMTP.From,
			To:      []string{u.Email},
			Subject: "Djinn - Verify email",
			Body:    fmt.Sprintf(verifyMail, webutil.BaseAddress(r), hex.EncodeToString(tok)),
		}

		if err := m.Send(h.SMTP.Client); err != nil {
			cause := errors.Cause(err)

			if rcpterrs, ok := cause.(*mail.ErrRcpts); ok {
				h.Log.Error.Println(r.Method, r.URL, "Failed to send verification email to "+rcpterrs.Error())
				goto resp
			}

			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to send verification email",
			}, "alert")
			h.RedirectBack(w, r)
			return
		}

	resp:
		sess.AddFlash(template.Alert{
			Level:   template.Success,
			Close:   true,
			Message: "Verification email sent to: "+u.Email,
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	b, err := hex.DecodeString(r.URL.Query().Get("token"))

	if err != nil {
		if errors.Cause(err) == user.ErrTokenExpired {
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Token expired, resend verification email",
			}, "alert")
			h.Redirect(w, r, "/settings")
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to verify account",
		}, "alert")
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
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to verify account",
		}, "alert")
		h.Redirect(w, r, "/settings")
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Account has been verified",
	}, "alert")
	h.Redirect(w, r, "/settings")
}

// Cleanup will either disable or enable artifact cleaning for the current user.
func (h User) Cleanup(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := r.ParseForm(); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to save changes",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	cleanup := r.PostForm.Get("cleanup") == "on"

	if err := h.Users.Update(u.ID, u.Email, cleanup, nil); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to save changes",
		}, "alert")
		h.Redirect(w, r, "/settings")
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Changes have been saved",
	}, "alert")
	h.RedirectBack(w, r)
}

// Email will update the user's email to the one given in the request.
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

	if err := webutil.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Success,
			Close:   true,
			Message: "Failed to update email",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	u.Email = f.Email
	u.UpdatedAt = time.Now()

	if err := h.Users.Update(u.ID, f.Email, u.Cleanup, []byte(f.VerifyPassword)); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Success,
			Close:   true,
			Message: "Failed to update email",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Email has been updated",
	}, "alert")
	h.RedirectBack(w, r)
}

// Password will update the user's password to the one given in the request.
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

	if err := webutil.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to update password",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := h.Users.Update(u.ID, u.Email, u.Cleanup, []byte(f.NewPassword)); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to update password",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Password has been updated",
	}, "alert")
	h.RedirectBack(w, r)
}

// Destroy will mark the user's account as deleted by setting the deleted_at
// column in the database to the current time.
func (h User) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	f := &user.DeleteForm{}

	if err := webutil.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to delete account",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	providers := provider.NewStore(h.DB, u)

	pp, err := providers.All()

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to delete account",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := providers.Delete(pp...); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to delete account",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := h.Users.Delete(u.ID, []byte(f.Password)); err != nil {
		cause := errors.Cause(err)

		if cause == user.ErrAuth {
			errs := webutil.NewErrors()
			errs.Put("delete_password", errors.New("Invalid password"))

			webutil.FlashFormWithErrors(sess, f, errs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to delete account",
		}, "alert")
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
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to delete account",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "user",
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Unix(0, 0),
	})
	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Account deleted",
	}, "alert")
	h.Redirect(w, r, "/")
}
