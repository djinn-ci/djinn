package user

import (
	"regexp"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"

	"github.com/andrewpillar/query"

	"golang.org/x/crypto/bcrypt"
)

type RegisterForm struct {
	Users *Store

	Email          string `schema:"email"`
	Username       string `schema:"username"`
	Password       string `schema:"password"`
	VerifyPassword string `schema:"verify_password"`
}

type LoginForm struct {
	Handle      string `schema:"handle"`
	Password    string `schema:"password"`
	RedirectURI string `schema:"redirect_uri"`
}

type EmailForm struct {
	User  *User  `schema:"-"`
	Users *Store `schema:"-"`

	Email          string `schema:"email"`
	VerifyPassword string `schema:"verify_password"`
}

type PasswordForm struct {
	User  *User  `schema:"-"`
	Users *Store `schema:"-"`

	OldPassword    string `schema:"old_password"`
	NewPassword    string `schema:"new_password"`
	VerifyPassword string `schema:"verify_password"`
}

type PasswordResetForm struct {
	Email string `schema:"email"`
}

type NewPasswordForm struct {
	Token          string `schema:"token"`
	Password       string `schema:"password"`
	VerifyPassword string `schema:"verify_password"`
}

type DeleteForm struct {
	Password string `schema:"delete_password"`
}

var (
	_ form.Form = (*RegisterForm)(nil)
	_ form.Form = (*LoginForm)(nil)
	_ form.Form = (*EmailForm)(nil)
	_ form.Form = (*PasswordForm)(nil)
	_ form.Form = (*PasswordResetForm)(nil)
	_ form.Form = (*NewPasswordForm)(nil)
	_ form.Form = (*DeleteForm)(nil)

	reEmail           = regexp.MustCompile("@")
	reAlphaNumDotDash = regexp.MustCompile("^[a-zA-Z0-9\\._\\-]+$")
)

// Fields returns a map of the Email and Username fields.
func (f RegisterForm) Fields() map[string]string {
	return map[string]string{
		"email":    f.Email,
		"username": f.Username,
	}
}

// Validate checks to see if the Email field is present and valid. An email is
// considered valid if it is less than 254 characters in length, and contains
// an @ character. The Username field is checked for presence, uniqueness, and
// validitity. A username must be between 3 and 64 characters, and only contain
// letters, numbers, dashes, and dots. The Password field is checked for
// presence, and length. It should be between 6 and 60 characters.
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
		query.Where("email", "=", query.Arg(f.Email)),
		query.OrWhere("username", "=", query.Arg(f.Username)),
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

// Fields returns a map containing the Handle field of the current
// LoginForm.
func (f LoginForm) Fields() map[string]string {
	return map[string]string{
		"handle": f.Handle,
	}
}

// Validate checks to see if the current LoginForm has a present Handle, and
// Password field.
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

// Fields returns a map containing the Email field of the current EmailForm.
func (f EmailForm) Fields() map[string]string {
	return map[string]string{
		"email": f.Email,
	}
}

// Validate checks to see if the Email field is present, valid, and unique. The
// uniqueness checks will be skipped if the email matches the existing one. The
// Password field is checked for presence, then compared against the one in the
// database for authentication.
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
		u, err := f.Users.Get(query.Where("email", "=", query.Arg(f.Email)))

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

// Fields will return an empty map of strings.
func (PasswordForm) Fields() map[string]string { return map[string]string{} }

// Validate the current PasswordForm. This checks for the presence of the
// old, new, and current password fields, as well as if the current password
// is valid based on what is in the database.
func (f PasswordForm) Validate() error {
	errs := form.NewErrors()

	if f.OldPassword == "" {
		errs.Put("old_password", form.ErrFieldRequired("Old password"))
	}

	if err := bcrypt.CompareHashAndPassword(f.User.Password, []byte(f.OldPassword)); err != nil {
		errs.Put("old_password", errors.New("Invalid password"))
	}

	if f.NewPassword == "" {
		errs.Put("new_password", form.ErrFieldRequired("New password"))
	}

	if len(f.NewPassword) < 6 || len(f.NewPassword) > 60 {
		errs.Put("new_password", form.ErrFieldInvalid("Password", "must be between 6 ans 60 characters in length"))
	}

	if f.VerifyPassword == "" {
		errs.Put("pass_verify_password", form.ErrFieldRequired("Password"))
	}

	if f.NewPassword != f.VerifyPassword {
		errs.Put("new_password", form.ErrFieldInvalid("Password", "does not match"))
		errs.Put("pass_verify_password", form.ErrFieldInvalid("Password", "does not match"))
	}
	return errs.Err()
}

func (f PasswordResetForm) Fields() map[string]string {
	return map[string]string{
		"email": f.Email,
	}
}

func (f PasswordResetForm) Validate() error {
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
	return errs.Err()
}

func (f NewPasswordForm) Fields() map[string]string { return map[string]string{} }

func (f NewPasswordForm) Validate() error {
	errs := form.NewErrors()

	if f.Password == "" {
		errs.Put("password", form.ErrFieldRequired("New password"))
	}

	if len(f.Password) < 6 || len(f.Password) > 60 {
		errs.Put("password", form.ErrFieldInvalid("Password", "must be between 6 ans 60 characters in length"))
	}

	if f.VerifyPassword == "" {
		errs.Put("verify_password", form.ErrFieldRequired("Password"))
	}

	if f.Password != f.VerifyPassword {
		errs.Put("password", form.ErrFieldInvalid("Password", "does not match"))
		errs.Put("verify_password", form.ErrFieldInvalid("Password", "does not match"))
	}
	return errs.Err()
}

func (f DeleteForm) Fields() map[string]string { return map[string]string{} }

func (f DeleteForm) Validate() error {
	errs := form.NewErrors()

	if f.Password == "" {
		errs.Put("delete_password", form.ErrFieldRequired("Password"))
	}
	return errs.Err()
}
