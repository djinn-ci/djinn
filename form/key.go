package form

import (
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"

	"golang.org/x/crypto/ssh"
)

type Key struct {
	Keys model.KeyStore `schema:"-"`
	Key  *model.Key     `schema:"-"`

	Namespace string `schema:"namespace"`
	Name      string `schema:"name"`
	Priv      string `schema:"key"`
	Config    string `schema:"config"`
}

func (f Key) Fields() map[string]string {
	m := make(map[string]string)
	m["namespace"] = f.Namespace
	m["name"] = f.Name
	m["key"] = f.Priv
	m["config"] = f.Config

	return m
}

func (f Key) Validate() error {
	errs := NewErrors()

	if f.Name == "" {
		errs.Put("name", ErrFieldRequired("Name"))
	}

	checkUnique := true

	if f.Key != nil && !f.Key.IsZero() {
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
			errs.Put("name", ErrFieldExists("Name"))
		}
	}

	if f.Priv == "" {
		errs.Put("key", ErrFieldRequired("Key"))
	}

	if _, err := ssh.ParsePrivateKey([]byte(f.Priv)); err != nil {
		errs.Put("key", ErrFieldInvalid("Key", err.Error()))
	}

	return errs.Err()
}
