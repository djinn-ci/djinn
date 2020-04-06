package user

import (
	"regexp"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"

	"github.com/andrewpillar/query"

	"golang.org/x/crypto/bcrypt"
)

type RegisterForm struct {
	Users Store

	Email          string `schema:"email"`
	Username       string `schema:"username"`
	Password       string `schema:"password"`
	VerifyPassword string `schema:"verify_password"`
}

type LoginForm struct {
	Handle   string `schema:"handle"`
	Password string `schema:"password"`
}

type EmailForm struct {
	User  *User `schema:"-"`
	Users Store `schema:"-"`

	Email          string `schema:"email"`
	VerifyPassword string `schema:"verify_password"`
}

type PasswordForm struct {
	User  *User `schema:"-"`
	Users Store `schema:"-"`

	OldPassword    string `schema:"old_password"`
	NewPassword    string `schema:"new_password"`
	VerifyPassword string `schema:"verify_password"`
}

var (
	_ form.Form = (*RegisterForm)(nil)
	_ form.Form = (*LoginForm)(nil)
	_ form.Form = (*EmailForm)(nil)
	_ form.Form = (*PasswordForm)(nil)

	reEmail           = regexp.MustCompile("@")
	reAlphaNumDotDash = regexp.MustCompile("^[a-zA-Z0-9\\._\\-]+$")
)

func (f RegisterForm) Fields() map[string]string {
	return map[string]string{
		"email":    f.Email,
		"username": f.Username,
	}
}

func (f RegisterForm) Validate() error {
	errs := form.NewErrors()

	if f.Email == "" {
		errs.Put("email", form.ErrFieldRequired("Email"))
	}

	if len(f.Email) > 254 {
		errs.Put("email", form.ErrFieldInvalid("Email", "must be less than 254 characters"))
	}

	if !reEmail.Match([]byte(f.Email)) {
		errs.Put("email", form.ErrFieldInvalid("Email", ""))
	}

	u, err := f.Users.Get(
		query.Where("email", "=", f.Email),
		query.OrWhere("username", "=", f.Username),
	)

	if err != nil {
		return errors.Err(err)
	}

	if !u.IsZero() {
		if f.Email == u.Email {
			errs.Put("email", form.ErrFieldExists("Email"))
		}

		if f.Username == u.Username {
			errs.Put("username", form.ErrFieldExists("Username"))
		}
	}

	if f.Username == "" {
		errs.Put("username", form.ErrFieldRequired("Username"))
	}

	if len(f.Username) < 3 || len(f.Username) > 64 {
		errs.Put("username", form.ErrFieldInvalid("Username", "must be between 3 and 32 characters in length"))
	}

	if !reAlphaNumDotDash.Match([]byte(f.Username)) {
		errs.Put("username", form.ErrFieldInvalid("Username", "can only contain letters, numbers, dashes, and dots"))
	}

	if f.Password == "" {
		errs.Put("password", form.ErrFieldRequired("Password"))
	}

	if len(f.Password) < 6 || len(f.Password) > 60 {
		errs.Put("password", form.ErrFieldInvalid("Password", "must be between 6 and 60 characters in length"))
	}

	if f.VerifyPassword == "" {
		errs.Put("verify_password", form.ErrFieldRequired("Verify password"))
	}

	if f.Password != f.VerifyPassword {
		errs.Put("password", form.ErrFieldInvalid("Password", "does not match"))
		errs.Put("verify_password", form.ErrFieldInvalid("Verify Password", "does not match"))
	}
	return errs.Err()
}

func (f LoginForm) Fields() map[string]string {
	return map[string]string{
		"handle": f.Handle,
	}
}

func (f LoginForm) Validate() error {
	errs := form.NewErrors()

	if f.Handle == "" {
		errs.Put("handle", form.ErrFieldRequired("Email or username"))
	}

	if f.Password == "" {
		errs.Put("password", form.ErrFieldRequired("Password"))
	}
	return errs.Err()
}

func (f EmailForm) Fields() map[string]string {
	return map[string]string{
		"email": f.Email,
	}
}

func (f EmailForm) Validate() error {
	errs := form.NewErrors()

	if f.Email == "" {
		errs.Put("email", form.ErrFieldRequired("Email"))
	}

	if len(f.Email) > 254 {
		errs.Put("email", form.ErrFieldInvalid("Email", "must be less than 254 characters"))
	}

	if !reEmail.Match([]byte(f.Email)) {
		errs.Put("email", form.ErrFieldInvalid("Email"))
	}

	if f.Email != f.User.Email {
		u, err := f.Users.Get(query.Where("email", "=", f.Email))

		if err != nil {
			return errors.Err(err)
		}

		if u.Email == f.Email {
			errs.Put("email", form.ErrFieldExists("Email"))
		}
	}

	if f.VerifyPassword == "" {
		errs.Put("email_verify_password", form.ErrFieldRequired("Password"))
	}

	if err := bcrypt.CompareHashAndPassword(f.User.Password, []byte(f.VerifyPassword)); err != nil {
		errs.Put("email_verify_password", errors.New("Invalid password"))
	}
	return errs.Err()
}

func (f PasswordForm) Fields() map[string]string { return map[string]string{} }

func (f PasswordForm) Validate() error {
	errs := form.NewErrors()

	if f.OldPassword == "" {
		errs.Put("old_password", form.ErrFieldRequired("Old Password"))
	}

	if err := bcrypt.CompareHashAndPassword(f.User.Password, []byte(f.OldPassword)); err != nil {
		errs.Put("old_password", errors.New("Invalid password"))
	}

	if len(f.NewPassword) < 6 || len(f.NewPassword) > 60 {
		errs.Put("new_password", form.ErrFieldInvalid("Password", "must be between 6 ans 60 characters in length"))
	}

	if f.VerifyPassword == "" {
		errs.Put("verify_password", form.ErrFieldRequired("Verify password"))
	}

	if f.NewPassword != f.VerifyPassword {
		errs.Put("new_password", form.ErrFieldInvalid("Password", "does not match"))
		errs.Put("pass_verify_password", form.ErrFieldInvalid("Password", "does not match"))
	}
	return errs.Err()
}
