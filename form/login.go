package form

import "github.com/andrewpillar/thrall/errors"

var ErrHandleRequired = errors.New("Email or username can't be blank")

type Login struct {
	Handle     string `schema:"handle"`
	Password   string `schema:"password"`
	RememberMe bool   `schema:"remember_me"`
}

func (f Login) Get(key string) string {
	if key == "handle" {
		return f.Handle
	}

	return ""
}

func (f Login) Validate() error {
	errs := NewErrors()

	if f.Handle == "" {
		errs.Put("handle", ErrHandleRequired)
	}

	if f.Password == "" {
		errs.Put("password", ErrPasswordRequired)
	}

	return errs.Final()
}
