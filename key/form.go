package key

import (
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/namespace"

	"github.com/andrewpillar/query"

	"golang.org/x/crypto/ssh"
)

// Form is the type that represents input data for adding a new SSH key.
type Form struct {
	namespace.Resource

	Key        *Key   `schema:"-"`
	Keys       *Store `schema:"-"`
	Name       string `schema:"name"   json:"name"`
	PrivateKey string `schema:"key"    json:"key"`
	Config     string `schema:"config" json:"config"`
}

var _ form.Form = (*Form)(nil)

// Fields returns a map containing the namespace, name, key, and config fields
// from the original Form.
func (f Form) Fields() map[string]string {
	return map[string]string{
		"namespace": f.Namespace,
		"name":      f.Name,
		"key":       f.PrivateKey,
		"config":    f.Config,
	}
}

// Validate checks to see if there is a name for the key, and if that name is
// unique to the namespace it is being added to or, to the user adding the key.
// It then checks to see if the key itself is a valid SSH private key.
func (f Form) Validate() error {
	errs := form.NewErrors()

	if err := f.Resource.BindNamespace(f.Keys); err != nil {
		return errors.Err(err)
	}

	if f.Name == "" && f.Key == nil {
		errs.Put("name", form.ErrFieldRequired("Name"))
	}

	checkUnique := true

	if f.Key != nil {
		if f.Key.Name == f.Name {
			checkUnique = false
		}
	}

	if checkUnique {
		opts := []query.Option{
			query.Where("name", "=", query.Arg(f.Name)),
		}

		if f.Keys.Namespace.IsZero() {
			opts = append(opts, query.Where("namespace_id", "IS", query.Lit("NULL")))
		}

		k, err := f.Keys.Get(opts...)

		if err != nil {
			return errors.Err(err)
		}

		if !k.IsZero() {
			errs.Put("name", form.ErrFieldExists("Name"))
		}
	}

	if f.PrivateKey == "" && f.Key == nil {
		errs.Put("key", form.ErrFieldRequired("Key"))
	}

	if f.Key == nil {
		if _, err := ssh.ParsePrivateKey([]byte(f.PrivateKey)); err != nil {
			errs.Put("key", form.ErrFieldInvalid("Key", err.Error()))
		}
	}
	return errs.Err()
}
