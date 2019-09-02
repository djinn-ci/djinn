package form

import (
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
)

type Object struct {
	Objects model.ObjectStore `schema:"-"`

	Writer  http.ResponseWriter   `schame:"-"`
	Request *http.Request         `schema:"-"`
	Limit   int64                 `schema:"-"`
	File    multipart.File        `schema:"-"`
	Info    *multipart.FileHeader `schema:"-"`

	Namespace string `schema:"namespace"`
	Name      string `schema:"name"`
}

func (f Object) Fields() map[string]string {
	m := make(map[string]string)
	m["namespace"] = f.Namespace
	m["name"] = f.Name

	return m
}

func (f *Object) Validate() error {
	errs := NewErrors()

	if f.Name == "" {
		errs.Put("name", ErrFieldRequired("Name"))
	}

	if !reAlphaNumDotDash.Match([]byte(f.Name)) {
		errs.Put("name", ErrFieldInvalid("Name", "can only contain letters, numbers, dashes, and dots"))
	}

	o, err := f.Objects.FindByName(f.Name)

	if err != nil {
		return errors.Err(err)
	}

	if !o.IsZero() {
		errs.Put("name", ErrFieldExists("Name"))
	}

	f.Request.Body = http.MaxBytesReader(f.Writer, f.Request.Body, f.Limit)

	if err := f.Request.ParseMultipartForm(f.Limit); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			errs.Put("file", ErrFieldInvalid("File", "too big"))
			return errs
		}

		return errors.Err(err)
	}

	f.File, f.Info, err = f.Request.FormFile("file")

	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			errs.Put("file", ErrFieldRequired("File"))
			return errs
		}

		errs.Put("file", ErrField("File", err))
	}

	return errs.Err()
}
