package form

import (
	"bytes"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
)

type Image struct {
	Upload `schema:"-"`

	Images     model.ImageStore `schema:"-"`
	Namespace  string           `schema:"namespace"`
	Name       string           `schema:"name"`
}

// QCOW magic number.
var magic = []byte{0x51, 0x46, 0x49, 0xFB}

func (f Image) Fields() map[string]string {
	return map[string]string{
		"name": f.Name,
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

	i, err := f.Images.FindByName(f.Name)

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
