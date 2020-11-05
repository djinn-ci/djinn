package image

import (
	"bytes"
	"regexp"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/namespace"

	"github.com/andrewpillar/query"
)

// Form is the type that represents input data for uploading a new driver image.
type Form struct {
	namespace.Resource
	form.File `schema:"-"`

	Images *Store `schema:"-"`
	Name   string `schema:"name"`
}

var (
	_ form.Form = (*Form)(nil)

	rename = regexp.MustCompile("^[a-zA-Z0-9_\\-]+$")
	magic  = []byte{0x51, 0x46, 0x49, 0xFB}
)

// Fields returns a map containing the namespace, and name fields from the
// original Form.
func (f *Form) Fields() map[string]string {
	return map[string]string{
		"namespace": f.Namespace,
		"name":      f.Name,
	}
}

// Validate checks to see if there is a name for the image, and if that name
// is valid. This will also check the contents of the uploaded file to make
// sure it is a valid QCOW2 image file. It does this by checking the first
// four bytes of the file match the ASCII magic number for QCOW2.
func (f *Form) Validate() error {
	errs := form.NewErrors()

	if err := f.Resource.BindNamespace(f.Images); err != nil {
		return errors.Err(err)
	}

	if f.Name == "" {
		errs.Put("name", form.ErrFieldRequired("Name"))
	}

	if !rename.Match([]byte(f.Name)) {
		errs.Put("name", form.ErrFieldInvalid("Name", "can only contain letters, numbers, and dashes"))
	}

	opts := []query.Option{
		query.Where("name", "=", f.Name),
	}

	if f.Images.Namespace.IsZero() {
		opts = append(opts, query.WhereRaw("namespace_id", "IS", "NULL"))
	}

	i, err := f.Images.Get(opts...)

	if err != nil {
		return errors.Err(err)
	}

	if !i.IsZero() {
		errs.Put("name", form.ErrFieldExists("Name"))
	}

	if err := f.File.Validate(); err != nil {
		if ferrs, ok := err.(form.Errors); ok {
			for k, v := range ferrs {
				for _, err := range v {
					errs.Put(k, errors.New(err))
				}
			}
			return errs.Err()
		}
		return errors.Err(err)
	}

	if f.File.File != nil {
		buf := make([]byte, 4, 4)

		if _, err := f.File.Read(buf); err != nil {
			return errors.Err(err)
		}

		if !bytes.Equal(buf, magic) {
			errs.Put("file", form.ErrFieldInvalid("File", "not a valid QCOW file format"))
		}

		f.File.Seek(0, 0)
	}
	return errs.Err()
}
