package form

import (
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
)

type Object struct {
	Upload `schema:"-"`

	Objects   model.ObjectStore `schema:"-"`
	Namespace string `schema:"namespace"`
	Name      string `schema:"name"`
}

func (f Object) Fields() map[string]string {
	m := make(map[string]string)
	m["namespace"] = f.Namespace
	m["name"] = f.Name

	return m
}

func (f *Object) Validate() error {
	errs := NewErrors()

	if f.Name == "" {
		errs.Put("name", ErrFieldRequired("Name"))
	}

	if !reAlphaNumDotDash.Match([]byte(f.Name)) {
		errs.Put("name", ErrFieldInvalid("Name", "can only contain letters, numbers, dashes, and dots"))
	}

	o, err := f.Objects.FindByName(f.Name)

	if err != nil {
		return errors.Err(err)
	}

	if !o.IsZero() {
		errs.Put("name", ErrFieldExists("Name"))
	}

	if err := f.Upload.Validate(); err != nil {
		for k, v := range err.(Errors) {
			for _, e := range v {
				errs.Put(k, errors.New(e))
			}
		}
	}

	return errs.Err()
}
