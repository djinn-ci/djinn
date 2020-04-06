package variable

import (
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/namespace"
)

type Form struct {
	namespace.ResourceForm

	Variables Store `schema:"-"`
	Key       string `schema:"key"`
	Value     string `schema:"value"`
}

var _ form.Form = (*Form)(nil)

func (f Form) Fields() map[string]string {
	return map[string]string{
		"key":   f.Key,
		"value": f.Value,
	}
}

func (f Form) Validate() error {
	errs := form.NewErrors()
	return errs.Err()
}
