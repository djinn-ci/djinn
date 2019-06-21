package form

import (
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
)

var (
	ErrFileRequired = errors.New("File is required")
	ErrFileTooLarge = errors.New("File is too big")
	ErrNameRequired = errors.New("Object name can't be blank")
)

type Object struct {
	Writer  http.ResponseWriter   `schame:"-"`
	Request *http.Request         `schema:"-"`
	Limit   int64                 `schema:"-"`
	Name    string                `schema:"name"`
	File    multipart.File        `schema:"-"`
	Info    *multipart.FileHeader `schema:"-"`
}

func (f *Object) Get(key string) string {
	if key == "name" {
		return f.Name
	}

	return ""
}

func (f *Object) Validate() error {
	errs := NewErrors()

	if f.Name == "" {
		errs.Put("name", ErrNameRequired)
	}

	f.Request.Body = http.MaxBytesReader(f.Writer, f.Request.Body, f.Limit)

	if err := f.Request.ParseMultipartForm(f.Limit); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			errs.Put("file", ErrFileTooLarge)
			return errs
		}

		log.Error.Println(errors.Err(err))
		errs.Put("file", errors.New("Failed to upload file"))

		return errs
	}

	var err error

	f.File, f.Info, err = f.Request.FormFile("file")

	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			errs.Put("file", ErrFileRequired)
			return errs
		}

		errs.Put("file", err)
	}

	return errs.Final()
}
