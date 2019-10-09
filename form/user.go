package form

import (
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"

	"golang.org/x/crypto/bcrypt"
)

type Email struct {
	User  *model.User
	Users model.UserStore

	Email          string `schema:"email"`
	VerifyPassword string `schema:"verify_password"`
}

type Password struct {
	User  *model.User
	Users model.UserStore

	OldPassword    string `schema:"old_password"`
	NewPassword    string `schema:"new_password"`
	VerifyPassword string `schema:"verify_password"`
}

func (f Email) Fields() map[string]string {
	return map[string]string{
		"email": f.Email,
	}
}

func (f Email) Validate() error {
	errs := NewErrors()

	if f.Email == "" {
		errs.Put("email", ErrFieldRequired("Email"))
	}

	if len(f.Email) > 254 {
		errs.Put("email", ErrFieldInvalid("Email", "must be less than 254 characters"))
	}

	if !reEmail.Match([]byte(f.Email)) {
		errs.Put("email", ErrFieldInvalid("Email", ""))
	}

	if f.Email != f.User.Email {
		u, err := f.Users.FindByEmail(f.Email)

		if err != nil {
			log.Error.Println(errors.Err(err))

			errs.Put("email", errors.Cause(err))
		} else {
			if u.Email == f.Email {
				errs.Put("email", ErrFieldExists("Email"))
			}
		}
	}

	if f.VerifyPassword == "" {
		errs.Put("email_verify_password", ErrFieldRequired("Password"))
	}

	if err := bcrypt.CompareHashAndPassword(f.User.Password, []byte(f.VerifyPassword)); err != nil {
		errs.Put("email_verify_password", errors.New("Password is not correct"))
	}

	return errs.Err()
}

func (f Password) Fields() map[string]string {
	return make(map[string]string)
}

func (f Password) Validate() error {
	errs := NewErrors()

	if f.OldPassword == "" {
		errs.Put("old_password", ErrFieldRequired("Old Password"))
	}

	if err := bcrypt.CompareHashAndPassword(f.User.Password, []byte(f.OldPassword)); err != nil {
		errs.Put("old_password", errors.New("Password is not correct"))
	}

	if len(f.NewPassword) < 6 || len(f.NewPassword) > 60 {
		errs.Put("new_password", ErrFieldInvalid("Password", "must be between 6 and 60 characters in length"))
	}

	if f.VerifyPassword == "" {
		errs.Put("verify_password", ErrFieldRequired("Verify Password"))
	}

	if f.NewPassword != f.VerifyPassword {
		errs.Put("new_password", ErrFieldInvalid("Password", "does not match"))
		errs.Put("pass_verify_password", ErrFieldInvalid("Verify Password", "does not match"))
	}

	return errs.Err()
}
