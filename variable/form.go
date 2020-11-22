package variable

import (
	"regexp"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

// Form is the type that represents input data for creating a new variable.
type Form struct {
	namespace.Resource

	Variables *Store `schema:"-"`
	Key       string `schema:"key"   json:"key"`
	Value     string `schema:"value" json:"value"`
}

var (
	_ webutil.Form = (*Form)(nil)

	revar = regexp.MustCompile("^[\\D]+[a-zA-Z0-9_]+$")
)

// Fields returns a map of fields for the current Form. This map will contain
// the Namespace, Key, and Value fields of the current Form.
func (f Form) Fields() map[string]string {
	return map[string]string{
		"namespace": f.Namespace,
		"key":       f.Key,
		"value":     f.Value,
	}
}

// Validate will bind a Namespace to the Form's Store, if the Namespace field
// is present. The presence of the Key field is then checked, followed by a
// validitity check for that Key (is only letters, and numbers with dashes). A
// uniqueness check is done on the Name for the current Namespace. Another check
// is also done to check the presence of the Value field.
func (f Form) Validate() error {
	errs := webutil.NewErrors()

	if err := f.Resource.Resolve(f.Variables); err != nil {
		return errors.Err(err)
	}

	if f.Key == "" {
		errs.Put("key", webutil.ErrFieldRequired("Key"))
	}

	opts := []query.Option{
		query.Where("key", "=", query.Arg(f.Key)),
	}

	if f.Variables.Namespace.IsZero() {
		opts = append(opts, query.Where("namespace_id", "IS", query.Lit("NULL")))
	}

	v, err := f.Variables.Get(opts...)

	if err != nil {
		return errors.Err(err)
	}

	if !v.IsZero() {
		errs.Put("key", webutil.ErrFieldExists("Key"))
	}

	if !revar.Match([]byte(f.Key)) {
		errs.Put("key", webutil.ErrField("Key", errors.New("invalid variable key")))
	}

	if f.Value == "" {
		errs.Put("value", webutil.ErrFieldRequired("Value"))
	}
	return errs.Err()
}
