package form

import (
	"regexp"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"
)

var reVariable = regexp.MustCompile("^[^0-9]+[a-zA-Z0-9_]+$")

type Variable struct {
	User      *model.User         `schema:"-"`
	Variables model.VariableStore `schema:"-"`

	Namespace string `schema:"namespace"`
	Key       string `schema:"key"`
	Value     string `schema:"value"`
}

func (f Variable) Fields() map[string]string {
	return map[string]string{
		"key":   f.Key,
		"value": f.Value,
	}
}

func (f Variable) Validate() error {
	errs := NewErrors()

	if f.Key == "" {
		errs.Put("key", ErrFieldRequired("Key"))
	}

	if !reVariable.Match([]byte(f.Key)) {
		errs.Put("key", ErrFieldInvalid("Key", "can only contain letters, numbers, dashes, and have not leading numbers"))
	}

	n, err := getNamespace(f.User, f.Namespace)

	if err != nil {
		return errors.Err(err)
	}

	if !n.IsZero() {
		f.Variables = n.VariableStore()
	}

	v, err := f.Variables.Get(query.Where("key", "=", f.Key))

	if err != nil {
		log.Error.Println(errors.Err(err))

		errs.Put("variable", errors.Cause(err))
	}

	if !v.IsZero() {
		errs.Put("key", ErrFieldExists("Key"))
	}

	if f.Value == "" {
		errs.Put("value", ErrFieldRequired("Value"))
	}

	return errs.Err()
}
