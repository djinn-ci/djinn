package form

import (
	"bytes"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"
)

type Image struct {
	Upload   `schema:"-"`

	User      *model.User      `schema:"-"`
	Images    model.ImageStore `schema:"-"`

	Namespace  string `schema:"namespace"`
	Name       string `schema:"name"`
}

// QCOW magic number.
var magic = []byte{0x51, 0x46, 0x49, 0xFB}

func (f Image) Fields() map[string]string {
	return map[string]string{
		"namespace": f.Namespace,
		"name":      f.Name,
	}
}

func (f *Image) Validate() error {
	errs := NewErrors()

	if f.Name == "" {
		errs.Put("name", ErrFieldRequired("Name"))
	}

	if !reAlphaNumDotDash.Match([]byte(f.Name)) {
		errs.Put("name", ErrFieldInvalid("Name", "can only contain letters, numbers, dashes, and dots"))
	}

	n, err := getNamespace(f.User, f.Namespace)

	if err != nil {
		return errors.Err(err)
	}

	if !n.IsZero() {
		f.Images = n.ImageStore()
	}

	i, err := f.Images.Get(query.Where("name", "=", f.Name))

	if err != nil {
		errs.Put("image", errors.Cause(err))
	}

	if !i.IsZero() {
		errs.Put("name", ErrFieldExists("Name"))
	}

	if err := f.Upload.Validate(); err != nil {
		for k, v := range err.(Errors) {
			for _, e := range v {
				errs.Put(k, errors.New(e))
			}
		}
	}

	if f.Upload.File != nil {
		buf := make([]byte, 4, 4)

		if _, err := f.Upload.File.Read(buf); err != nil {
			errs.Put("image", err)
		}

		if !bytes.Equal(buf, magic) {
			errs.Put("file", ErrFieldInvalid("File", "not a valid QCOW file format"))
		}

		if _, err := f.Upload.File.Read(buf); err != nil {
			errs.Put("image", err)
		}

		f.Upload.File.Seek(0, 0)
	}

	return errs.Err()
}
