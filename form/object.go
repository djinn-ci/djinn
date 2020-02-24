package form

import (
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"
)

type Object struct {
	Upload `schema:"-"`

	User    *model.User       `schema:"-"`
	Objects model.ObjectStore `schema:"-"`

	Namespace string `schema:"namespace"`
	Name      string `schema:"name"`
}

func (f Object) Fields() map[string]string {
	return map[string]string{
		"namespace": f.Namespace,
		"name":      f.Name,
	}

}

func (f *Object) Validate() error {
	errs := NewErrors()

	if f.Name == "" {
		errs.Put("name", ErrFieldRequired("Name"))
	}

	if !reAlphaNumDotDash.Match([]byte(f.Name)) {
		errs.Put("name", ErrFieldInvalid("Name", "can only contain letters, numbers, dashes, and dots"))
	}

	n, err := getNamespace(f.User, f.Namespace)

	if err != nil {
		return errors.Err(err)
	}

	if !n.IsZero() {
		f.Objects = n.ObjectStore()
	}

	o, err := f.Objects.Get(query.Where("name", "=", f.Name))

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
