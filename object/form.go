package object

import (
	"regexp"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/namespace"

	"github.com/andrewpillar/query"
)

// Form is the type that represents input data for uploading a new object.
type Form struct {
	namespace.Resource
	form.File `schema:"-"`

	Objects *Store `schema:"-"`
	Name    string `schema:"name"`
}

var (
	_ form.Form = (*Form)(nil)

	rename = regexp.MustCompile("^[a-zA-Z0-9\\._\\-]+$")
)

// Fields returns a map of fields for the current Form. This map will contain
// the Namespace, and Name fields of the current Form.
func (f Form) Fields() map[string]string {
	return map[string]string{
		"namespace": f.Namespace,
		"name":      f.Name,
	}
}

// Validate will bind a Namespace to the Form's Store, if the Namespace field
// is present. The presence of the Name field is then checked, followed by a
// validity check for that Name (is only letters, numbers, dashes, and dots). A
// uniqueness check on the Name is then done for the current Namespace.
func (f *Form) Validate() error {
	errs := form.NewErrors()

	if err := f.Resource.BindNamespace(f.Objects); err != nil {
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
		ferrs, ok := err.(form.Errors)

		if !ok {
			return errors.Err(err)
		}

		for k, v := range ferrs {
			for _, err := range v {
				errs.Put(k, errors.New(err))
			}
		}
	}
	return errs.Err()
}
