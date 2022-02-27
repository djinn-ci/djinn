package http

import (
	"djinn-ci.com/errors"
	"djinn-ci.com/key"
	"djinn-ci.com/namespace"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"golang.org/x/crypto/ssh"
)

type Form struct {
	Namespace namespace.Path
	Name      string
	Key       string
	Config    string
}

var _ webutil.Form = (*Form)(nil)

func (f Form) Fields() map[string]string {
	return map[string]string{
		"namespace": f.Namespace.String(),
		"name":      f.Name,
		"key":       f.Key,
		"config":    f.Config,
	}
}

type Validator struct {
	UserID int64
	Keys   *key.Store
	Key    *key.Key
	Form   Form
}

func (v *Validator) Validate(errs webutil.ValidationErrors) {
	if v.Key != nil {
		if v.Form.Name == "" {
			v.Form.Name = v.Key.Name
		}
		if v.Form.Config == "" {
			v.Form.Config = v.Key.Config
		}
	}

	if v.Form.Name == "" {
		errs.Add("name", webutil.ErrFieldRequired("Name"))
	}

	if v.Key == nil || (v.Key != nil && v.Form.Name != v.Key.Name) {
		opts := []query.Option{
			query.Where("user_id", "=", query.Arg(v.UserID)),
			query.Where("name", "=", query.Arg(v.Form.Name)),
		}

		if v.Form.Namespace.Valid {
			_, n, err := v.Form.Namespace.ResolveOrCreate(v.Keys.Pool, v.UserID)

			if err != nil {
				if perr, ok := err.(*namespace.PathError); ok {
					errs.Add("namespace", perr)
					return
				}
				errs.Add("fatal", err)
				return
			}

			if err := n.IsCollaborator(v.Keys.Pool, v.UserID); err != nil {
				if errors.Is(err, namespace.ErrPermission) {
					errs.Add("namespace", err)
					return
				}
				errs.Add("fatal", err)
				return
			}
			opts[0] = query.Where("namespace_id", "=", query.Arg(n.ID))
		}

		_, ok, err := v.Keys.Get(opts...)

		if err != nil {
			errs.Add("key", err)
			return
		}

		if ok {
			errs.Add("key", webutil.ErrFieldExists("Name"))
		}
	}

	if v.Key == nil && v.Form.Key == "" {
		errs.Add("key", webutil.ErrFieldRequired("Key"))
		return
	}

	if v.Key == nil {
		// Only validate on key creation, since the key itself cannot be updated
		// once created.
		if _, err := ssh.ParsePrivateKey([]byte(v.Form.Key)); err != nil {
			errs.Add("key", err)
		}
	}
}
