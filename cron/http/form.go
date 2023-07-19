package http

import (
	"context"

	"djinn-ci.com/auth"
	"djinn-ci.com/cron"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/manifest"
	"djinn-ci.com/namespace"

	"github.com/andrewpillar/webutil/v2"
)

type Form struct {
	Pool *database.Pool `schema:"-"`
	User *auth.User     `schema:"-"`
	Cron *cron.Cron     `schema:"-"`

	Name     string
	Schedule cron.Schedule
	Manifest manifest.Manifest
}

func (f *Form) Fields() map[string]string {
	return map[string]string{
		"name":     f.Name,
		"schedule": f.Schedule.String(),
		"manifest": f.Manifest.String(),
	}
}

func (f *Form) Validate(ctx context.Context) error {
	var v webutil.Validator

	v.WrapError(
		webutil.IgnoreError("name", auth.ErrPermission),
		webutil.MapError(auth.ErrPermission, errors.New("cannot submit to namespace")),
		webutil.WrapFieldError,
	)

	if f.Cron != nil {
		if f.Name == "" {
			f.Name = f.Cron.Name
		}
		if f.Schedule == 0 {
			f.Schedule = f.Cron.Schedule
		}
		if f.Manifest.String() == "" {
			f.Manifest = f.Cron.Manifest
		}
	}

	v.Add("name", f.Name, webutil.FieldRequired)

	v.Add("manifest", f.Manifest.Namespace, func(ctx context.Context, val any) error {
		p, err := namespace.ParsePath(val.(string))

		if err != nil {
			return err
		}
		return namespace.CanAccess(f.Pool, f.User)(ctx, p)
	})
	v.Add("manifest", f.Manifest, webutil.FieldRequired)
	v.Add("manifest", f.Manifest, func(ctx context.Context, val any) error {
		m := val.(manifest.Manifest)
		return m.Validate()
	})

	var pathError error

	if f.Cron == nil || (f.Cron != nil && f.Name != f.Cron.Name) {
		path, err := namespace.ParsePath(f.Manifest.Namespace)

		if err != nil {
			pathError = err
		} else {
			v.Add("name", f.Name, namespace.ResourceUnique[*cron.Cron](cron.NewStore(f.Pool), f.User, "name", path))
		}
	}

	errs := v.Validate(ctx)

	if pathError != nil {
		errs.Add("namespace", &webutil.FieldError{
			Name: "Namespace",
			Err:  pathError,
		})
	}
	return errs.Err()
}
