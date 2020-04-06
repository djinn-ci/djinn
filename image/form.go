package image

import (
	"bytes"
	"regexp"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/namespace"

	"github.com/andrewpillar/query"
)

type Form struct {
	namespace.ResourceForm
	form.File `schema:"-"`

	Images Store  `schema:"-"`
	Name   string `schema:"name"`
}

var (
	_ form.Form = (*Form)(nil)

	rename = regexp.MustCompile("^[a-zA-Z0-9\\._\\-]+$")
	magic  = []byte{0x51, 0x46, 0x49, 0xFB}
)

func (f Form) Fields() map[string]string {
	return map[string]string{
		"namespace": f.Namespace,
		"name":      f.Name,
	}
}

func (f *Form) Validate() error {
	errs := form.NewErrors()

	if err := f.ResourceForm.Validate(); err != nil {
		return errors.Err(err)
	}

	if f.Name == "" {
		errs.Put("name", form.ErrFieldRequired("Name"))
	}

	if !rename.Match([]byte(f.Name)) {
		errs.Put("name", form.ErrFieldInvalid("Name", "can only contain letters, numbers, dashes, and dots"))
	}

	i, err := f.Images.Get(query.Where("name", "=", f.Name))

	if err != nil {
		return errors.Err(err)
	}

	if !i.IsZero() {
		errs.Put("name", form.ErrFieldExists("Name"))
	}

	if err := f.File.Validate(); err != nil {
		for k, v := range err.(form.Errors) {
			for _, err := range v {
				errs.Put(k, errors.New(err))
			}
		}
	}

	if f.File.File != nil {
		buf := make([]byte, 0, 4)

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
