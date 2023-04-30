package http

import (
	"context"
	"regexp"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/webutil/v2"
)

type Form struct {
	Pool *database.Pool `schema:"-"`
	User *auth.User     `schema:"-"`

	Namespace namespace.Path
	Key       string
	Value     string
	Mask      bool
}

var _ webutil.Form = (*Form)(nil)

func (f Form) Fields() map[string]string {
	masktab := map[bool]string{
		true:  "true",
		false: "false",
	}

	return map[string]string{
		"namespace": f.Namespace.String(),
		"key":       f.Key,
		"value":     f.Value,
		"mask":      masktab[f.Mask],
	}
}

var reKey = regexp.MustCompile("^[\\D]+[a-zA-Z0-9_]+$")

func (f Form) Validate(ctx context.Context) error {
	var v webutil.Validator

	v.WrapError(
		webutil.IgnoreError("key", database.ErrPermission),
		webutil.MapError(database.ErrPermission, errors.New("permission denied")),
		webutil.WrapFieldError,
	)

	v.Add("namespace", f.Namespace, namespace.CanAccess(f.Pool, f.User))

	v.Add("key", f.Key, webutil.FieldRequired)
	v.Add("key", f.Key, webutil.FieldMatches(reKey))
	v.Add("key", f.Key, namespace.ResourceUnique[*variable.Variable](variable.NewStore(f.Pool), f.User, "key", f.Namespace))

	v.Add("value", f.Value, webutil.FieldRequired)

	if f.Mask {
		v.Add("value", f.Value, webutil.FieldMinLen(variable.MaskLen))
	}

	errs := v.Validate(ctx)

	return errs.Err()
}
