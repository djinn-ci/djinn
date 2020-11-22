package cron

import (
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/manifest"
	"github.com/andrewpillar/djinn/namespace"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

// Form is the type that represents input data for creating and editing a cron
// job.
type Form struct {
	namespace.Resource

	Crons    *Store            `schema:"-"`
	Cron     *Cron             `schema:"-"`
	Name     string            `schema:"name"`
	Schedule Schedule          `schema:"schedule"`
	Manifest manifest.Manifest `schema:"manifest"`
}

var _ webutil.Form = (*Form)(nil)

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
func (f Form) Validate() error {
	errs := webutil.NewErrors()

	if err := f.Resource.Resolve(f.Crons); err != nil {
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
		errs.Put("name", webutil.ErrFieldRequired("Name"))
	}

	checkUnique := true

	if !f.Cron.IsZero() {
		checkUnique = f.Cron.Name != f.Name
	}

	if checkUnique {
		opts := []query.Option{
			query.Where("name", "=", query.Arg(f.Name)),
		}

		if f.Crons.Namespace.IsZero() {
			opts = append(opts, query.Where("namespace_id", "IS", query.Lit("NULL")))
		}

		c, err := f.Crons.Get(opts...)

		if err != nil {
			return errors.Err(err)
		}

		if !c.IsZero() {
			errs.Put("name", webutil.ErrFieldExists("Name"))
		}
	}

	if f.Manifest.String() == "{}" {
		errs.Put("manifest", webutil.ErrFieldRequired("Manifest"))
	}

	if err := f.Manifest.Validate(); err != nil {
		errs.Put("manifest", webutil.ErrField("Build manifest", err))
	}
	return errs.Err()
}
