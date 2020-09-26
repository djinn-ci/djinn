package cron

import (
	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/namespace"

	"github.com/andrewpillar/query"
)

// Form is the type that represents input data for creating and editing a cron
// job.
type Form struct {
	namespace.Resource

	Crons    *Store          `schema:"-"`
	Cron     *Cron           `schema:"-"`
	Name     string          `schema:"name"`
	Schedule Schedule        `schema:"schedule"`
	Manifest config.Manifest `schema:"manifest"`
}

var _ form.Form = (*Form)(nil)

// Fields returns a map of fields for the current Form. This map will contain
// the Namespace, Name, Schedule, and Manifest fields of the current form.
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

// Validate will bind a Namespace to the Form's Store, if the Namespace field
// is present. The presence of the Name field is then checked, followed by a
// uniqueness check of that field. The present oft he Manifest field is then
// checked, followed by a validation of that manifest.
func (f *Form) Validate() error {
	errs := form.NewErrors()

	if err := f.Resource.BindNamespace(f.Crons); err != nil {
		return errors.Err(err)
	}

	if f.Cron != nil {
		if f.Name == "" {
			f.Name = f.Cron.Name
		}

		if f.Schedule == Schedule(0) {
			f.Schedule = f.Cron.Schedule
		}

		if f.Manifest.String() == "{}" {
			f.Manifest = f.Cron.Manifest
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
