package http

import (
	"context"
	"regexp"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/user"

	"github.com/andrewpillar/webutil/v2"
)

type RegisterForm struct {
	DB *database.Pool `schema:"-"`

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

var (
	reEmail    = regexp.MustCompile("@")
	reUsername = regexp.MustCompile("^[a-zA-Z0-9.-]+$")
)

func (f RegisterForm) Validate(ctx context.Context) error {
	var v webutil.Validator

	nametab := map[string]string{
		"email":           "Email",
		"username":        "Username",
		"password":        "Password",
		"verify_password": "Password",
	}

	v.WrapError(func(name string, err error) error {
		if _, ok := err.(webutil.MatchError); ok {
			switch name {
			case "email":
				err = errors.New("invalid")
			case "username":
				err = errors.New("can only have letters, numbers, dots, and dashes")
			}
		}
		if s, ok := nametab[name]; ok {
			name = s
		}
		if errors.Is(err, database.ErrExists) {
			err = webutil.ErrFieldExists
		}
		return webutil.WrapFieldError(name, err)
	})

	users := user.NewStore(f.DB)

	v.Add("email", f.Email, webutil.FieldRequired)
	v.Add("email", f.Email, webutil.FieldMatches(reEmail))
	v.Add("email", f.Email, database.FieldUnique[*auth.User](users, "email"))

	v.Add("username", f.Username, webutil.FieldRequired)
	v.Add("username", f.Username, webutil.FieldMatches(reUsername))
	v.Add("username", f.Username, webutil.FieldLen(3, 30))
	v.Add("username", f.Username, database.FieldUnique[*auth.User](users, "username"))

	v.Add("password", f.Password, webutil.FieldRequired)
	v.Add("password", f.Password, webutil.FieldLen(6, 60))

	v.Add("verify_password", f.VerifyPassword, webutil.FieldRequired)
	v.Add("verify_password", f.VerifyPassword, webutil.FieldEquals(f.Password))

	errs := v.Validate(ctx)

	return errs.Err()
}

type LoginForm struct {
	AuthMech    string `schema:"auth_mech"`
	Handle      string
	Password    string
	RedirectURI string `schema:"redirect_uri"`
}

func (f LoginForm) Fields() map[string]string {
	return map[string]string{
		"handle": f.Handle,
	}
}

func (f LoginForm) Validate(ctx context.Context) error {
	if f.AuthMech != user.InternalProvider {
		return nil
	}

	var v webutil.Validator

	nametab := map[string]string{
		"handle":   "Email or username",
		"password": "Password",
	}

	v.WrapError(func(name string, err error) error {
		if s, ok := nametab[name]; ok {
			name = s
		}
		return webutil.WrapFieldError(name, err)
	})

	v.WrapError(func(name string, err error) error {
		if s, ok := nametab[name]; ok {
			name = s
		}
		return webutil.WrapFieldError(name, err)
	})

	v.Add("handle", f.Handle, webutil.FieldRequired)
	v.Add("password", f.Password, webutil.FieldRequired)

	errs := v.Validate(ctx)

	return errs.Err()
}

type EmailForm struct {
	DB *database.Pool `schema:"-"`

	Token          string
	Email          string `schema:"update_email.email"`
	VerifyPassword string `schema:"update_email.verify_password"`
	RedirectURI    string `schema:"redirect_uri"`
}

var _ webutil.Form = (*EmailForm)(nil)

func (f EmailForm) Fields() map[string]string {
	return map[string]string{"email": f.Email}
}

func (f EmailForm) Validate(ctx context.Context) error {
	var v webutil.Validator

	v.WrapError(
		webutil.MapError(database.ErrExists, errors.New("already exists")),
		webutil.WrapFieldError,
	)

	v.Add("update_email.email", f.Email, webutil.FieldRequired)
	v.Add("update_email.email", f.Email, webutil.FieldMatches(reEmail))
	v.Add("update_email.email", f.Email, database.FieldUniqueExcept[*auth.User](user.NewStore(f.DB), "email", f.Email))

	v.Add("update_email.verify_password", f.VerifyPassword, webutil.FieldRequired)

	errs := v.Validate(ctx)

	return errs.Err()
}

type PasswordForm struct {
	User *auth.User `schema:"-"`

	OldPassword    string `schema:"update_password.old_password"`
	NewPassword    string `schema:"update_password.new_password"`
	VerifyPassword string `schema:"update_password.verify_new_password"`
}

var _ webutil.Form = (*PasswordForm)(nil)

func (f PasswordForm) Fields() map[string]string { return nil }

func (f PasswordForm) Validate(ctx context.Context) error {
	var v webutil.Validator

	nametab := map[string]string{
		"update_password.old_password":        "Password",
		"update_password.new_password":        "Password",
		"update_password.verify_new_password": "Password",
	}

	v.WrapError(func(name string, err error) error {
		if s, ok := nametab[name]; ok {
			name = s
		}
		return webutil.WrapFieldError(name, err)
	})

	v.Add("update_password.old_password", f.OldPassword, webutil.FieldRequired)
	v.Add("update_password.old_password", f.OldPassword, user.ValidatePassword(f.User))

	v.Add("update_password.new_password", f.NewPassword, webutil.FieldRequired)
	v.Add("update_password.new_password", f.NewPassword, webutil.FieldLen(6, 60))

	v.Add("update_password.verify_new_password", f.VerifyPassword, webutil.FieldRequired)
	v.Add("update_password.verify_new_password", f.VerifyPassword, webutil.FieldEquals(f.NewPassword))

	errs := v.Validate(ctx)

	return errs.Err()
}

type PasswordResetForm struct {
	Email string
}

var _ webutil.Form = (*PasswordResetForm)(nil)

func (f PasswordResetForm) Fields() map[string]string {
	return map[string]string{"email": f.Email}
}

func (f PasswordResetForm) Validate(ctx context.Context) error {
	var v webutil.Validator

	v.Add("email", f.Email, webutil.FieldRequired)
	v.Add("email", f.Email, webutil.FieldMatches(reEmail))

	errs := v.Validate(ctx)

	return errs.Err()
}

type NewPasswordForm struct {
	Token          string
	Password       string
	VerifyPassword string `schema:"verify_password"`
}

var _ webutil.Form = (*NewPasswordForm)(nil)

func (f NewPasswordForm) Fields() map[string]string { return nil }

func (f NewPasswordForm) Validate(ctx context.Context) error {
	var v webutil.Validator

	nametab := map[string]string{
		"password":        "Password",
		"verify_password": "Password",
	}

	v.WrapError(func(name string, err error) error {
		if s, ok := nametab[name]; ok {
			name = s
		}
		return webutil.WrapFieldError(name, err)
	})

	v.Add("password", f.Password, webutil.FieldRequired)
	v.Add("password", f.Password, webutil.FieldLen(6, 60))

	v.Add("verify_password", f.VerifyPassword, webutil.FieldRequired)
	v.Add("verify_password", f.VerifyPassword, webutil.FieldEquals(f.VerifyPassword))

	errs := v.Validate(ctx)

	return errs.Err()
}

type DeleteForm struct {
	User     *auth.User `schema:"-"`
	Password string     `schema:"delete_account.verify_password"`
}

var _ webutil.Form = (*DeleteForm)(nil)

func (f DeleteForm) Fields() map[string]string { return nil }

func (f DeleteForm) Validate(ctx context.Context) error {
	var v webutil.Validator

	nametab := map[string]string{
		"delete_account.verify_password": "Password",
	}

	v.WrapError(func(name string, err error) error {
		if s, ok := nametab[name]; ok {
			name = s
		}
		return webutil.WrapFieldError(name, err)
	})

	v.Add("delete_account.verify_password", f.Password, webutil.FieldRequired)
	v.Add("delete_account.verify_password", f.Password, user.ValidatePassword(f.User))

	errs := v.Validate(ctx)

	return errs.Err()
}

type SudoForm struct {
	User  *auth.User `schema:"-"`
	Token string     `schema:"-"`

	SudoToken   string `schema:"sudo_token"`
	SudoURL     string `schema:"sudo_url"`
	SudoReferer string `schema:"sudo_referer"`
	Password    string
}

var _ webutil.Form = (*SudoForm)(nil)

func (f SudoForm) Fields() map[string]string { return nil }

func (f SudoForm) Validate(ctx context.Context) error {
	var v webutil.Validator

	nametab := map[string]string{
		"password": "Password",
	}

	v.WrapError(func(name string, err error) error {
		if s, ok := nametab[name]; ok {
			name = s
		}
		return webutil.WrapFieldError(name, err)
	})

	v.Add("password", f.Password, webutil.FieldRequired)
	v.Add("password", f.Password, user.ValidatePassword(f.User))

	v.Add("sudo_token", f.SudoToken, webutil.FieldEquals(f.Token))

	errs := v.Validate(ctx)

	return errs.Err()
}
