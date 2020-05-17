package form

import (
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/andrewpillar/thrall/errors"
)

// File provides an implementation of the Form interface to validate file
// uploads via HTTP. It embeds the underlying multipart.File type from the
// stdlib.
type File struct {
	multipart.File

	// Writer is the ResponseWriter to which the HTTP response will be written.
	Writer http.ResponseWriter

	// Request is the current HTTP request through which the file is being
	// uploaded.
	Request *http.Request

	// Limit specifies the maximum size of the file to be uploaded.
	Limit int64

	// Info is the header of the uploaded file.
	Info *multipart.FileHeader

	// Disallowed is a list of MIME types that are not considered to be valid.
	Disallowed []string
}

// Fields is a stub method to statisfy the Form interface, and returns only
// an empty map.
func (f File) Fields() map[string]string { return map[string]string{} }

// Validate checks to see if a file was uploaded, is within the given limit,
// and if the MIME type of the file is valid.
func (f *File) Validate() error {
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
