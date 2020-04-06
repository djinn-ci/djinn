package key

import (
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/namespace"

	"github.com/andrewpillar/query"

	"golang.org/x/crypto/ssh"
)

type Form struct {
	namespace.ResourceForm

	Key        *Key   `schema:"-"`
	Keys       Store  `schema:"-"`
	Name       string `schema:"name"`
	PrivateKey string `schema:"key"`
	Config     string `schema:"config"`
}

var _ form.Form = (*Form)(nil)

func (f Form) Fields() map[string]string {
	return map[string]string{
		"namespace": f.Namespace,
		"name":      f.Name,
		"key":       f.PrivateKey,
		"config":    f.Config,
	}
}

func (f Form) Validate() error {
	errs := form.NewErrors()

	if f.Name == "" {
		errs.Put("name", form.ErrFieldRequired("Name"))
	}

	if err := f.ResourceForm.Validate(); err != nil {
		return errors.Err(err)
	}

	checkUnique := true

	if f.Key != nil {
		if f.Key.Name == f.Name {
			checkUnique = false
		}
	}

	if checkUnique {
		k, err := f.Keys.Get(query.Where("name", "=", f.Name))

		if err != nil {
			return errors.Err(err)
		}

		if !k.IsZero() {
			errs.Put("name", form.ErrFieldExists("Name"))
		}
	}

	if f.PrivateKey == "" {
		errs.Put("key", form.ErrFieldRequired("Key"))
	}

	if _, err := ssh.ParsePrivateKey([]byte(f.PrivateKey)); err != nil {
		errs.Put("key", form.ErrFieldInvalid("Key", err.Error()))
	}
	return errs.Err()
}
