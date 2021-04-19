package object

import (
	"regexp"

	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

// Form is the type that represents input data for uploading a new object.
type Form struct {
	namespace.Resource
	*webutil.File

	Objects *Store `schema:"-"`
	Name    string `schema:"name"`
}

var (
	_ webutil.Form = (*Form)(nil)

	rename = regexp.MustCompile("^[a-zA-Z0-9\\._\\-]+$")
)

// Fields returns a map of fields for the current Form. This map will contain
// the Namespace, and Name fields of the current Form.
func (f Form) Fields() map[string]string {
	return map[string]string{
		"namespace": f.Namespace,
		"name":      f.Name,
	}
}

// Validate will bind a Namespace to the Form's Store, if the Namespace field
// is present. The presence of the Name field is then checked, followed by a
// validity check for that Name (is only letters, numbers, dashes, and dots). A
// uniqueness check on the Name is then done for the current Namespace.
func (f *Form) Validate() error {
	errs := webutil.NewErrors()

	if err := f.Resource.Resolve(f.Objects); err != nil {
		return errors.Err(err)
	}

	if f.Name == "" {
		errs.Put("name", webutil.ErrFieldRequired("Name"))
	}

	if !rename.Match([]byte(f.Name)) {
		errs.Put("name", webutil.ErrField("Name", errors.New("can only contain letters, numbers, dashes, and dots")))
	}

	opts := []query.Option{
		query.Where("name", "=", query.Arg(f.Name)),
	}

	if f.Objects.Namespace.IsZero() {
		opts = append(opts, query.Where("namespace_id", "IS", query.Lit("NULL")))
	}

	o, err := f.Objects.Get(opts...)

	if err != nil {
		return errors.Err(err)
	}

	if !o.IsZero() {
		errs.Put("name", webutil.ErrFieldExists("Name"))
	}

	if err := f.File.Validate(); err != nil {
		ferrs, ok := err.(*webutil.Errors)

		if !ok {
			return errors.Err(err)
		}
		errs.Merge(ferrs)
	}
	return errs.Err()
}
