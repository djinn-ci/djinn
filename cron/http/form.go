package http

import (
	"djinn-ci.com/cron"
	"djinn-ci.com/manifest"
	"djinn-ci.com/namespace"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

type Form struct {
	Name     string
	Schedule cron.Schedule
	Manifest manifest.Manifest
}

func (f Form) Fields() map[string]string {
	return map[string]string{
		"name":     f.Name,
		"schedule": f.Schedule.String(),
		"manifest": f.Manifest.String(),
	}
}

type Validator struct {
	UserID int64
	Crons  cron.Store
	Cron   *cron.Cron
	Form   Form
}

func (v *Validator) Validate(errs webutil.ValidationErrors) {
	if v.Cron != nil {
		if v.Form.Name == "" {
			v.Form.Name = v.Cron.Name
		}
		if v.Form.Schedule == cron.Schedule(0) {
			v.Form.Schedule = v.Cron.Schedule
		}
		if v.Form.Manifest.String() == "{}" {
			v.Form.Manifest = v.Cron.Manifest
		}
	}

	if v.Form.Name == "" {
		errs.Add("name", webutil.ErrFieldRequired("Name"))
	}

	if v.Cron == nil || (v.Cron != nil && v.Form.Name != v.Cron.Name) {
		opts := []query.Option{
			query.Where("user_id", "=", query.Arg(v.UserID)),
			query.Where("name", "=", query.Arg(v.Form.Name)),
		}

		if v.Form.Manifest.Namespace != "" {
			path, err := namespace.ParsePath(v.Form.Manifest.Namespace)

			if err != nil {
				errs.Add("fatal", err)
				return
			}

			_, n, err := path.ResolveOrCreate(v.Crons.Pool, v.UserID)

			if err != nil {
				errs.Add("fatal", err)
				return
			}
			opts[0] = query.Where("namespace_id", "=", query.Arg(n.ID))
		}

		_, ok, err := v.Crons.Get(opts...)

		if err != nil {
			errs.Add("fatal", err)
			return
		}

		if ok {
			errs.Add("name", webutil.ErrFieldExists("Name"))
		}
	}

	if v.Form.Manifest.String() == "{}" {
		errs.Add("manifest", webutil.ErrFieldRequired("Manifest"))
	}

	if err := v.Form.Manifest.Validate(); err != nil {
		errs.Add("manifest", err)
	}
}
