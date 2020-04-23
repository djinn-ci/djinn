package object

import (
	"regexp"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/namespace"

	"github.com/andrewpillar/query"
)

type Form struct {
	namespace.Resource
	form.File `schema:"-"`

	Objects Store  `schema:"-"`
	Name    string `schema:"name"`
}

var (
	_ form.Form = (*Form)(nil)

	rename = regexp.MustCompile("^[a-zA-Z0-9\\._\\-]+$")
)

func (f Form) Fields() map[string]string {
	return map[string]string{
		"namespace": f.Namespace,
		"name":      f.Name,
	}
}

func (f Form) Validate() error {
	errs := form.NewErrors()

	if err := f.Resource.BindNamespace(&f.Objects); err != nil {
		return errors.Err(err)
	}

	if f.Name == "" {
		errs.Put("name", form.ErrFieldRequired("Name"))
	}

	if !rename.Match([]byte(f.Name)) {
		errs.Put("name", form.ErrFieldInvalid("Name", "can only contain letters, numbers, dashes, and dots"))
	}

	o, err := f.Objects.Get(query.Where("name", "=", f.Name))

	if err != nil {
		return errors.Err(err)
	}

	if !o.IsZero() {
		errs.Put("name", form.ErrFieldExists("Name"))
	}

	if err := f.File.Validate(); err != nil {
		for k, v := range err.(form.Errors) {
			for _, err := range v {
				errs.Put(k, errors.New(err))
			}
		}
	}
	return errs.Err()
}
