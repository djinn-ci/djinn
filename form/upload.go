package form

import (
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/andrewpillar/thrall/errors"
)

type Upload struct {
	Writer     http.ResponseWriter   `schema:"-"`
	Request    *http.Request         `schema:"-"`
	Limit      int64                 `schema:"-"`
	File       multipart.File        `schema:"-"`
	Info       *multipart.FileHeader `schema:"-"`
	Disallowed []string              `schema:"-"`
}

func (f Upload) Fields() map[string]string {
	return make(map[string]string)
}

func (f *Upload) Validate() error {
	errs := NewErrors()

	if f.Limit > 0 {
		f.Request.Body = http.MaxBytesReader(f.Writer, f.Request.Body, f.Limit)
	}

	if err := f.Request.ParseMultipartForm(f.Limit); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			errs.Put("file", ErrFieldInvalid("File", "too big"))
		} else {
			errs.Put("file", errors.New("Failed to upload file: " + err.Error()))
		}
	}

	var err error

	f.File, f.Info, err = f.Request.FormFile("file")

	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			errs.Put("file", ErrFieldRequired("File"))
		} else {
			errs.Put("file", errors.New("Failed to upload file: " + err.Error()))
		}
	}

	if len(f.Disallowed) > 0 {
		for _, mime := range f.Disallowed {
			if f.Info.Header.Get("Content-Type") == mime {
				errs.Put("file", ErrFieldInvalid("File", "cannot be a ZIP file or equivalent"))
			}
		}
	}

	return errs.Err()
}
