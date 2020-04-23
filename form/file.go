package form

import (
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/andrewpillar/thrall/errors"
)

type File struct {
	multipart.File

	Writer     http.ResponseWriter
	Request    *http.Request
	Limit      int64
	Info       *multipart.FileHeader
	Disallowed []string
}

func (f File) Fields() map[string]string { return map[string]string{} }

func (f File) Validate() error {
	errs := NewErrors()

	if f.Limit > 0 {
		f.Request.Body = http.MaxBytesReader(f.Writer, f.Request.Body, f.Limit)
	}

	if err := f.Request.ParseMultipartForm(f.Limit); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			errs.Put("file", ErrFieldInvalid("File", "too big"))
		}
		return errors.Err(err)
	}

	var err error

	f.File, f.Info, err = f.Request.FormFile("file")

	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			errs.Put("file", ErrFieldRequired("File"))
			return errs.Err()
		}
		return errors.Err(err)
	}

	header := make([]byte, 512)

	if _, err := f.Read(header); err != nil {
		return errors.Err(err)
	}

	if _, err := f.Seek(0, 0); err != nil {
		return errors.Err(err)
	}

	if len(f.Disallowed) > 0 && f.Info != nil {
		disallowed := strings.Join(f.Disallowed, ", ")

		for _, mime := range f.Disallowed {
			if http.DetectContentType(header) == mime {
				errs.Put("file", ErrFieldInvalid("File", "cannot be one of " + disallowed))
			}
		}
	}
	return errs.Err()
}
