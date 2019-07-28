package form

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
		errs.Put("handle", ErrFieldRequired("Email or username"))
	}

	if f.Password == "" {
		errs.Put("password", ErrFieldRequired("Password"))
	}

	return errs.Err()
}
