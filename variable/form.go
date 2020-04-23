package variable

import (
	"regexp"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/namespace"

	"github.com/andrewpillar/query"
)

type Form struct {
	namespace.Resource

	Variables Store `schema:"-"`
	Key       string `schema:"key"`
	Value     string `schema:"value"`
}

var (
	_ form.Form = (*Form)(nil)

	revar = regexp.MustCompile("^[\\D]+[a-zA-Z0-9_]+$")
)

func (f Form) Fields() map[string]string {
	return map[string]string{
		"key":   f.Key,
		"value": f.Value,
	}
}

func (f Form) Validate() error {
	errs := form.NewErrors()

	if err := f.Resource.BindNamespace(&f.Variables); err != nil {
		return errors.Err(err)
	}

	if f.Key == "" {
		errs.Put("key", form.ErrFieldRequired("Key"))
	}

	v, err := f.Variables.Get(query.Where("key", "=", f.Key))

	if err != nil {
		return errors.Err(err)
	}

	if !v.IsZero() {
		errs.Put("key", form.ErrFieldExists("Key"))
	}

	if !revar.Match([]byte(f.Key)) {
		errs.Put("key", form.ErrFieldInvalid("Key", "invalid variable key"))
	}

	if f.Value == "" {
		errs.Put("value", form.ErrFieldRequired("Value"))
	}
	return errs.Err()
}
