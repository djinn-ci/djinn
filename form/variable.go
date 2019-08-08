package form

import (
	"regexp"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
)

var reVariable = regexp.MustCompile("^[^0-9]+[a-zA-Z0-9_]+$")

type Variable struct {
	Variables model.VariableStore `schema:"-"`

	Namespace string `schema:"namespace"`
	Key       string `schema:"key"`
	Value     string `schema:"value"`
}

func (f Variable) Fields() map[string]string {
	m := make(map[string]string)
	m["key"] = f.Key
	m["value"] = f.Value

	return m
}

func (f Variable) Validate() error {
	errs := NewErrors()

	if f.Key == "" {
		errs.Put("key", ErrFieldRequired("Key"))
	}

	if !reVariable.Match([]byte(f.Key)) {
		errs.Put("key", ErrFieldInvalid("Key", "can only contain letters, numbers, dashes, and have not leading numbers"))
	}

	v, err := f.Variables.FindByKey(f.Key)

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
