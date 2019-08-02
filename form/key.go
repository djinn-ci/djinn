package form

import (
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"

	"golang.org/x/crypto/ssh"
)

type Key struct {
	Keys model.KeyStore `schema:"-"`

	Name   string `schema:"name"`
	Key    string `schema:"key"`
	Config string `schema:"config"`
}

func (f Key) Fields() map[string]string {
	m := make(map[string]string)
	m["name"] = f.Name
	m["key"] = f.Key
	m["config"] = f.Config

	return m
}

func (f Key) Validate() error {
	errs := NewErrors()

	if f.Name == "" {
		errs.Put("name", ErrFieldRequired("Name"))
	}

	k, err := f.Keys.FindByName(f.Name)

	if err != nil {
		log.Error.Println(errors.Err(err))

		errs.Put("key", errors.Cause(err))
	}

	if !k.IsZero() {
		errs.Put("name", ErrFieldExists("Name"))
	}

	if f.Key == "" {
		errs.Put("key", ErrFieldRequired("Key"))
	}

	if _, err := ssh.ParsePrivateKey([]byte(f.Key)); err != nil {
		errs.Put("key", ErrFieldInvalid("Key", err.Error()))
	}

	return errs.Err()
}
