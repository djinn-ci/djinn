package http

import (
	"context"
	"mime/multipart"
	"regexp"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/object"

	"github.com/andrewpillar/webutil/v2"
)

type Form struct {
	Pool *database.Pool `schema:"-"`
	User *auth.User     `schema:"-"`

	Namespace namespace.Path
	File      multipart.File
	Name      string
}

var _ webutil.Form = (*Form)(nil)

func (f *Form) Fields() map[string]string {
	return map[string]string{
		"namespace": f.Namespace.String(),
		"name":      f.Name,
	}
}

var reName = regexp.MustCompile("^[a-zA-Z0-9\\._\\-]+$")

func (f *Form) Validate(ctx context.Context) error {
	var v webutil.Validator

	v.WrapError(
		webutil.IgnoreError("name", auth.ErrPermission),
		webutil.MapError(auth.ErrPermission, errors.New("permission denied")),
		webutil.WrapFieldError,
	)

	v.Add("namespace", f.Namespace, namespace.CanAccess(f.Pool, f.User))

	v.Add("name", f.Name, webutil.FieldRequired)
	v.Add("name", f.Name, webutil.FieldMatches(reName))
	v.Add("name", f.Name, namespace.ResourceUnique[*object.Object](object.NewStore(f.Pool), f.User, "name", f.Namespace))

	v.Add("file", f.File, func(_ context.Context, val any) error {
		if val == nil {
			return webutil.ErrFieldRequired
		}
		return nil
	})

	errs := v.Validate(ctx)

	return errs.Err()
}
