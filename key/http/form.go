package http

import (
	"context"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/key"
	"djinn-ci.com/namespace"

	"github.com/andrewpillar/webutil/v2"

	"golang.org/x/crypto/ssh"
)

type Form struct {
	Pool *database.Pool `schema:"-"`
	User *auth.User     `schema:"-"`
	Key  *key.Key       `schema:"-"`

	Namespace namespace.Path
	Name      string
	SSHKey    string `json:"key" schema:"key"`
	Config    string
}

func (f *Form) Fields() map[string]string {
	return map[string]string{
		"namespace": f.Namespace.String(),
		"name":      f.Name,
		"key":       f.SSHKey,
		"config":    f.Config,
	}
}

func (f *Form) Validate(ctx context.Context) error {
	var v webutil.Validator

	v.WrapError(
		webutil.IgnoreError("name", auth.ErrPermission),
		webutil.MapError(auth.ErrPermission, errors.New("permission denied")),
		webutil.WrapFieldError,
	)

	if f.Key != nil {
		if f.Name == "" {
			f.Name = f.Key.Name
		}
		if f.Config == "" {
			f.Config = f.Key.Config
		}
	}

	v.Add("namespace", f.Namespace, namespace.CanAccess(f.Pool, f.User))

	v.Add("name", f.Name, webutil.FieldRequired)

	if f.Key == nil || (f.Key != nil && f.Name != f.Key.Name) {
		v.Add("name", f.Name, namespace.ResourceUnique[*key.Key](key.NewStore(f.Pool), f.User, "name", f.Namespace))
	}

	if f.Key == nil {
		v.Add("key", f.SSHKey, webutil.FieldRequired)
		v.Add("key", f.SSHKey, func(_ context.Context, val any) error {
			_, err := ssh.ParsePrivateKey([]byte(val.(string)))
			return err
		})
	}

	errs := v.Validate(ctx)

	return errs.Err()
}
