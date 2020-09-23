package cron

import (
	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/namespace"

	"github.com/andrewpillar/query"
)

type Form struct {
	namespace.Resource

	Crons    *Store          `schema:"-"`
	Cron     *Cron           `schema:"-"`
	Name     string          `schema:"name"`
	Schedule Schedule        `schema:"schedule"`
	Manifest config.Manifest `schema:"manifest"`
}

var _ form.Form = (*Form)(nil)

func (f Form) Fields() map[string]string {
	manifest := f.Manifest.String()

	if manifest == "{}" {
		manifest = ""
	}

	return map[string]string{
		"namespace": f.Namespace,
		"name":      f.Name,
		"schedule":  f.Schedule.String(),
		"manifest":  manifest,
	}
}

func (f Form) Validate() error {
	errs := form.NewErrors()

	if f.Cron != nil {
		if f.Name == "" {
			f.Name = f.Cron.Name
		}
	}

	if f.Name == "" {
		errs.Put("name", form.ErrFieldRequired("Name"))
	}

	checkUnique := true

	if !f.Cron.IsZero() {
		checkUnique = f.Cron.Name != f.Name
	}

	if checkUnique {
		c, err := f.Crons.Get(query.Where("name", "=", f.Name))

		if err != nil {
			return errors.Err(err)
		}

		if !c.IsZero() {
			errs.Put("name", form.ErrFieldExists("Name"))
		}
	}

	if f.Manifest.String() == "{}" {
		errs.Put("manifest", form.ErrFieldRequired("Manifest"))
	}

	if err := f.Manifest.Validate(); err != nil {
		errs.Put("manifest", form.ErrFieldInvalid("Build manifest", err.Error()))
	}
	return errs.Err()
}
