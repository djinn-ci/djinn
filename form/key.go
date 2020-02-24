package form

import (
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"

	"golang.org/x/crypto/ssh"
)

type Key struct {
	User *model.User    `schema:"-"`
	Key  *model.Key     `schema:"-"`
	Keys model.KeyStore `schema:"-"`

	Namespace string `schema:"namespace"`
	Name      string `schema:"name"`
	Priv      string `schema:"key"`
	Config    string `schema:"config"`
}

func (f Key) Fields() map[string]string {
	return map[string]string{
		"namespace": f.Namespace,
		"name":      f.Name,
		"key":       f.Priv,
		"config":    f.Config,
	}
}

func (f Key) Validate() error {
	errs := NewErrors()

	if f.Name == "" {
		errs.Put("name", ErrFieldRequired("Name"))
	}

	n, err := getNamespace(f.User, f.Namespace)

	if err != nil {
		return errors.Err(err)
	}

	if !n.IsZero() {
		f.Keys = n.KeyStore()
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
