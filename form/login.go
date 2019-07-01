package form

import "github.com/andrewpillar/thrall/errors"

var ErrHandleRequired = errors.New("Email or username can't be blank")

type Login struct {
	Handle     string `schema:"handle"`
	Password   string `schema:"password"`
	RememberMe bool   `schema:"remember_me"`
}

func (f Login) Fields() map[string]string {
	m := make(map[string]string)
	m["handle"] = f.Handle

	return m
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
