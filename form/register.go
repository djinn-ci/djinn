package form

import (
	"regexp"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"
)

var reEmail = regexp.MustCompile("@")

type Register struct {
	Users model.UserStore

	Email          string `schema:"email"`
	Username       string `schema:"username"`
	Password       string `schema:"password"`
	VerifyPassword string `schema:"verify_password"`
}

func (f Register) Fields() map[string]string {
	m := make(map[string]string)
	m["email"] = f.Email
	m["username"] = f.Username

	return m
}

func (f Register) Validate() error {
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

	u, err := f.Users.Get(query.Where("email", "=", f.Email))

	if err != nil {
		log.Error.Println(errors.Err(err))

		errs.Put("register", errors.Cause(err))
	} else if !u.IsZero() {
		errs.Put("email", ErrFieldExists("Email"))
	}

	if f.Username == "" {
		errs.Put("username", ErrFieldRequired("Username"))
	}

	if len(f.Username) < 3 || len(f.Username) > 64 {
		errs.Put("username", ErrFieldInvalid("Username", "must be between 3 and 32 characters in length"))
	}

	if !reAlphaNumDotDash.Match([]byte(f.Username)) {
		errs.Put("username", ErrFieldInvalid("Username", "can only contain letters, numbers, dashes, and dots"))
	}

	u, err = f.Users.Get(query.Where("username", "=", f.Username))

	if err != nil {
		log.Error.Println(errors.Err(err))

		errs.Put("register", errors.Cause(err))
	} else if !u.IsZero() {
		errs.Put("username", ErrFieldExists("Username"))
	}

	if f.Password == "" {
		errs.Put("password", ErrFieldRequired("Password"))
	}

	if len(f.Password) < 6 || len(f.Password) > 60 {
		errs.Put("password", ErrFieldInvalid("Password", "must be between 6 and 60 characters in length"))
	}

	if f.VerifyPassword == "" {
		errs.Put("verify_password", ErrFieldRequired("Verify password"))
	}

	if f.Password != f.VerifyPassword {
		errs.Put("password", ErrFieldInvalid("Password", "does not match"))
		errs.Put("verify_password", ErrFieldInvalid("Verify Password", "does not match"))
	}

	return errs.Err()
}
