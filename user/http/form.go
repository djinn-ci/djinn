package http

import (
	"regexp"

	"djinn-ci.com/errors"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"golang.org/x/crypto/bcrypt"
)

type RegisterForm struct {
	Email          string
	Username       string
	Password       string
	VerifyPassword string `schema:"verify_password"`
}

var _ webutil.Form = (*RegisterForm)(nil)

func (f RegisterForm) Fields() map[string]string {
	return map[string]string{
		"email":    f.Email,
		"username": f.Username,
	}
}

type RegisterValidator struct {
	Users user.Store
	Form  RegisterForm
}

var (
	_ webutil.Validator = (*RegisterValidator)(nil)

	reemail = regexp.MustCompile("@")
)

func (v RegisterValidator) Validate(errs webutil.ValidationErrors) {
	if v.Form.Email == "" {
		errs.Add("email", webutil.ErrFieldRequired("Email"))
	}

	if len(v.Form.Email) > 254 {
		errs.Add("email", errors.New("Email must be less than 254 characters in length"))
	}

	if !reemail.Match([]byte(v.Form.Email)) {
		errs.Add("email", errors.New("Invalid email address"))
	}

	u, ok, err := v.Users.Get(
		query.Where("email", "=", query.Arg(v.Form.Email)),
		query.OrWhere("username", "=", query.Arg(v.Form.Username)),
	)

	if err != nil {
		errs.Add("fatal", err)
		return
	}

	if ok {
		if v.Form.Email == u.Email {
			errs.Add("email", webutil.ErrFieldExists("Email"))
		}
		if v.Form.Username == u.Username {
			errs.Add("username", webutil.ErrFieldExists("Username"))
		}
	}

	if v.Form.Username == "" {
		errs.Add("username", webutil.ErrFieldRequired("Username"))
	}

	if len(v.Form.Username) < 3 || len(v.Form.Username) > 32 {
		errs.Add("username", errors.New("Username must be between 3 and 32 characters in length"))
	}

	if v.Form.Password == "" {
		errs.Add("password", webutil.ErrFieldRequired("Password"))
	}

	if len(v.Form.Password) < 6 || len(v.Form.Password) > 60 {
		errs.Add("password", errors.New("Password must be between 6 and 60 characters in length"))
	}

	if v.Form.VerifyPassword == "" {
		errs.Add("verify_password", webutil.ErrFieldRequired("Verify password"))
	}

	if v.Form.Password != v.Form.VerifyPassword {
		errs.Add("password", errors.New("Password does not match"))
		errs.Add("verify_password", errors.New("Password does not match"))
	}
}

type LoginForm struct {
	Handle      string
	Password    string
	RedirectURI string `schema:"redirect_uri"`
}

var (
	_ webutil.Form      = (*LoginForm)(nil)
	_ webutil.Validator = (*LoginForm)(nil)
)

func (f LoginForm) Fields() map[string]string {
	return map[string]string{
		"handle": f.Handle,
	}
}

func (f LoginForm) Validate(errs webutil.ValidationErrors) {
	if f.Handle == "" {
		errs.Add("handle", webutil.ErrFieldRequired("Email or username"))
	}

	if f.Password == "" {
		errs.Add("password", webutil.ErrFieldRequired("Password"))
	}
}

type EmailForm struct {
	Token          string
	Email          string
	VerifyPassword string `schema:"verify_password"`
	RedirectURI    string `schema:"redirect_uri"`
}

var _ webutil.Form = (*EmailForm)(nil)

func (f EmailForm) Fields() map[string]string {
	return map[string]string{
		"email": f.Email,
	}
}

type EmailValidator struct {
	Users user.Store
	User  *user.User
	Form  EmailForm
}

func (v EmailValidator) Validate(errs webutil.ValidationErrors) {
	if v.Form.Email == "" {
		errs.Add("email", webutil.ErrFieldRequired("Email"))
	}

	if len(v.Form.Email) > 254 {
		errs.Add("email", errors.New("must be less than 254 characters"))
	}

	if !reemail.Match([]byte(v.Form.Email)) {
		errs.Add("email", errors.New("Invalid email address"))
	}

	if v.Form.Email != v.User.Email {
		_, ok, err := v.Users.Get(query.Where("email", "=", query.Arg(v.Form.Email)))

		if err != nil {
			errs.Add("fatal", err)
			return
		}

		if ok {
			errs.Add("email", webutil.ErrFieldExists("Email"))
		}
	}

	if v.Form.VerifyPassword == "" {
		if v.Form.Token == "" {
			errs.Add("email_verify_password", webutil.ErrFieldRequired("Password"))
		}
	}
}

type PasswordForm struct {
	OldPassword    string `schema:"old_password"`
	NewPassword    string `schema:"new_password"`
	VerifyPassword string `schema:"verify_password"`
}

var _ webutil.Form = (*PasswordForm)(nil)

func (f PasswordForm) Fields() map[string]string { return nil }

type PasswordValidator struct {
	User  *user.User
	Users user.Store
	Form  PasswordForm
}

func (v PasswordValidator) Validate(errs webutil.ValidationErrors) {
	if v.Form.OldPassword == "" {
		errs.Add("old_password", webutil.ErrFieldRequired("Old password"))
	}

	if err := bcrypt.CompareHashAndPassword(v.User.Password, []byte(v.Form.OldPassword)); err != nil {
		errs.Add("old_password", errors.New("Invalid password"))
	}

	if v.Form.NewPassword == "" {
		errs.Add("new_password", webutil.ErrFieldRequired("New password"))
	}

	if len(v.Form.NewPassword) < 6 || len(v.Form.NewPassword) > 60 {
		errs.Add("new_password", errors.New("Password must be between 6 and 60 characters in length"))
	}

	if v.Form.VerifyPassword == "" {
		errs.Add("pass_verify_password", webutil.ErrFieldRequired("Password"))
	}

	if v.Form.NewPassword != v.Form.VerifyPassword {
		errs.Add("new_password", errors.New("Password does not match"))
		errs.Add("pass_verify_password", errors.New("Password does not match"))
	}
}

type PasswordResetForm struct {
	Email string
}

var (
	_ webutil.Form      = (*PasswordResetForm)(nil)
	_ webutil.Validator = (*PasswordResetForm)(nil)
)

func (f PasswordResetForm) Fields() map[string]string {
	return map[string]string{"email": f.Email}
}

func (f PasswordResetForm) Validate(errs webutil.ValidationErrors) {
	if f.Email == "" {
		errs.Add("email", webutil.ErrFieldRequired("Email"))
	}

	if len(f.Email) > 254 {
		errs.Add("email", errors.New("Email must be less than 254 characters in length"))
	}

	if !reemail.Match([]byte(f.Email)) {
		errs.Add("email", errors.New("Invalid email address"))
	}
}

type NewPasswordForm struct {
	Token          string
	Password       string
	VerifyPassword string `schema:"verify_password"`
}

var (
	_ webutil.Form      = (*NewPasswordForm)(nil)
	_ webutil.Validator = (*NewPasswordForm)(nil)
)

func (f NewPasswordForm) Fields() map[string]string { return nil }

func (f NewPasswordForm) Validate(errs webutil.ValidationErrors) {
	if f.Password == "" {
		errs.Add("password", webutil.ErrFieldRequired("New password"))
	}

	if len(f.Password) < 6 || len(f.Password) > 60 {
		errs.Add("password", errors.New("Password must be between 6 and 60 characters in length"))
	}

	if f.VerifyPassword == "" {
		errs.Add("verify_password", webutil.ErrFieldRequired("Password"))
	}

	if f.Password != f.VerifyPassword {
		errs.Add("password", errors.New("Password does not match"))
		errs.Add("verify_password", errors.New("Password does not match"))
	}
}

type DeleteForm struct {
	Password string `schema:"delete_password"`
}

var _ webutil.Form = (*DeleteForm)(nil)

func (f DeleteForm) Fields() map[string]string { return nil }

type DeleteValidator struct {
	User *user.User
	Form DeleteForm
}

var _ webutil.Validator = (*DeleteValidator)(nil)

func (v DeleteValidator) Validate(errs webutil.ValidationErrors) {
	if v.Form.Password == "" {
		errs.Add("delete_password", webutil.ErrFieldRequired("Password"))
	}

	if err := bcrypt.CompareHashAndPassword(v.User.Password, []byte(v.Form.Password)); err != nil {
		errs.Add("delete_password", errors.New("Invalid password"))
	}
}
