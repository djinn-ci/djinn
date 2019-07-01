package form

import (
	"regexp"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
)

var (
	emailPattern = "@"
	emailRegex   = regexp.MustCompile(emailPattern)

	usernamePattern = "^[_-a-zA-Z0-9\\S.]+$"
	usernameRegex   = regexp.MustCompile(usernamePattern)

	ErrEmailRequired = errors.New("Email can't be blank")
	ErrEmailLen      = errors.New("Email must be less than 254 characters")
	ErrEmailInvalid  = errors.New("Email is not valid")
	ErrEmailTaken    = errors.New("Email is already taken")

	ErrUsernameRequired = errors.New("Username can't be blank")
	ErrUsernameLen      = errors.New("Username must be between 3 and 32 characters")
	ErrUsernameInvalid  = errors.New("Username can only contain letters, numbers, dashes, and dots")
	ErrUsernameTaken    = errors.New("Username is already taken")

	ErrPasswordRequired = errors.New("Password can't be blank")
	ErrPasswordLen      = errors.New("Password must be between 6 and 60 characters")
	ErrPasswordMismatch = errors.New("Passwords do not match")
)

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
		errs.Put("email", ErrEmailRequired)
	}

	if len(f.Email) > 254 {
		errs.Put("email", ErrEmailLen)
	}

	if !emailRegex.Match([]byte(f.Email)) {
		errs.Put("email", ErrEmailInvalid)
	}

	u, err := f.Users.FindByEmail(f.Email)

	if err != nil {
		log.Error.Println(errors.Err(err))

		errs.Put("register", errors.Cause(err))
	} else if !u.IsZero() {
		errs.Put("email", ErrEmailTaken)
	}

	if f.Username == "" {
		errs.Put("username", ErrUsernameRequired)
	}

	if len(f.Username) < 3 || len(f.Username) > 64 {
		errs.Put("username", ErrUsernameLen)
	}

	if !usernameRegex.Match([]byte(f.Username)) {
		errs.Put("username", ErrUsernameInvalid)
	}

	u, err = f.Users.FindByUsername(f.Username)

	if err != nil {
		log.Error.Println(errors.Err(err))

		errs.Put("register", errors.Cause(err))
	} else if !u.IsZero() {
		errs.Put("username", ErrUsernameTaken)
	}

	if f.Password == "" {
		errs.Put("password", ErrPasswordRequired)
	}

	if len(f.Password) < 6 || len(f.Password) > 60 {
		errs.Put("password", ErrPasswordLen)
	}

	if f.VerifyPassword == "" {
		errs.Put("verify_password", ErrPasswordRequired)
	}

	if f.Password != f.VerifyPassword {
		errs.Put("password", ErrPasswordMismatch)
		errs.Put("verify_password", ErrPasswordMismatch)
	}

	return errs.Final()
}
