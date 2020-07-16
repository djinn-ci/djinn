package form

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/andrewpillar/thrall/errors"
)

type sectionReadCloser struct {
	*io.SectionReader
}

// File provides an implementation of the Form interface to validate file
// uploads via HTTP. It embeds the underlying multipart.File type from the
// stdlib. If the given HTTP request is of multipart/form-data then the request
// body will be parsed for the form, otherwise the entire request body is
// treated as the file's content.
type File struct {
	multipart.File

	// Writer is the ResponseWriter to which the HTTP response will be written.
	Writer http.ResponseWriter

	// Request is the current HTTP request through which the file is being
	// uploaded.
	Request *http.Request

	// Limit specifies the maximum size of the file to be uploaded.
	Limit int64

	// MIMEType is the media type of the file being uploaded.
	MIMEType string

	// Disallowed is a list of MIME types that are not considered to be valid.
	Disallowed []string
}

// Fields is a stub method to statisfy the Form interface, and returns only
// an empty map.
func (f *File) Fields() map[string]string { return map[string]string{} }

// Validate checks to see if a file was uploaded, is within the given limit,
// and if the MIME type of the file is valid.
func (f *File) Validate() error {
	errs := NewErrors()

	if f.Limit > 0 {
		f.Request.Body = http.MaxBytesReader(f.Writer, f.Request.Body, f.Limit)
	}

	var err error

	if strings.HasPrefix(f.Request.Header.Get("Content-Type"), "multipart/form-data") {
		if err := f.Request.ParseMultipartForm(f.Limit); err != nil {
			if strings.Contains(err.Error(), "request body too large") {
				errs.Put("file", ErrFieldInvalid("File", "too big"))
			}
			return errors.Err(err)
		}

		f.File, _, err = f.Request.FormFile("file")
	} else {
		buf := &bytes.Buffer{}

		if _, err := io.Copy(buf, f.Request.Body); err != nil {
			if strings.Contains(err.Error(), "request body too large") {
				errs.Put("file", ErrFieldInvalid("File", "too big"))
			}
			return errors.Err(err)
		}

		b := buf.Bytes()

		f.File = sectionReadCloser{
			SectionReader: io.NewSectionReader(bytes.NewReader(b), 0, int64(len(b))),
		}
	}

	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			errs.Put("file", ErrFieldRequired("File"))
			return errs.Err()
		}
		return errors.Err(err)
	}

	header := make([]byte, 512)

	if _, err := f.Read(header); err != nil {
		if err == io.EOF {
			errs.Put("file", ErrFieldRequired("File"))
		} else {
			return errors.Err(err)
		}
	}

	if _, err := f.Seek(0, 0); err != nil {
		return errors.Err(err)
	}

	f.MIMEType = http.DetectContentType(header)

	if len(f.Disallowed) > 0 {
		disallowed := strings.Join(f.Disallowed, ", ")

		for _, mime := range f.Disallowed {
			if f.MIMEType == mime {
				errs.Put("file", ErrFieldInvalid("File", "cannot be one of " + disallowed))
			}
		}
	}
	return errs.Err()
}

func (rc sectionReadCloser) Close() error { return nil }
