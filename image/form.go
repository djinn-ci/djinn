package image

import (
	"bytes"
	"io"
	"regexp"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

// Form is the type that represents input data for uploading a new driver image.
type Form struct {
	namespace.Resource
	*webutil.File

	Images *Store `schema:"-"`
	Name   string `schema:"name"`
}

var (
	_ webutil.Form = (*Form)(nil)

	rename = regexp.MustCompile("^[a-zA-Z0-9_\\-]+$")
	magic  = []byte{0x51, 0x46, 0x49, 0xFB} // QCOW file format magic number
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
	errs := webutil.NewErrors()

	if err := f.Resource.Resolve(f.Images); err != nil {
		return errors.Err(err)
	}

	if f.Name == "" {
		errs.Put("name", webutil.ErrFieldRequired("Name"))
	}

	if !rename.Match([]byte(f.Name)) {
		errs.Put("name", webutil.ErrField("Name", errors.New("can only contain letters, numbers, and dashes")))
	}

	opts := []query.Option{
		query.Where("name", "=", query.Arg(f.Name)),
	}

	if f.Images.Namespace.IsZero() {
		opts = append(opts, query.Where("namespace_id", "IS", query.Lit("NULL")))
	}

	i, err := f.Images.Get(opts...)

	if err != nil {
		return errors.Err(err)
	}

	if !i.IsZero() {
		errs.Put("name", webutil.ErrFieldExists("Name"))
	}

	if err := f.File.Validate(); err != nil {
		if ferrs, ok := err.(*webutil.Errors); ok {
			errs.Merge(ferrs)
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
			errs.Put("file", webutil.ErrField("File", errors.New("not a valid QCOW file format")))
		}
		f.File.Seek(0, io.SeekStart)
	}
	return errs.Err()
}
