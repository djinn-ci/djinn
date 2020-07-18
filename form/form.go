// Package form provides an interface and functions for working with the
// unmarshalling and validating of HTTP form data.
package form

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/andrewpillar/thrall/errors"

	"github.com/gorilla/schema"
)

type Form interface {
	// Validate the given form and return any errors that occur. If validation
	// fails then it is expected for the returned error to be of type
	// form.Errors.
	Validate() error

	// Fields returns a map of all the fields in the form, and their values.
	Fields() map[string]string
}

// ErrField returns an error for a form field that encountered a generic
// underlying error.
func ErrField(field string, err error) error {
	err = errors.Cause(err)
	return errors.New(field + " unexpected error: " + err.Error())
}

// ErrFieldExists returns an error for a form field whose value already exists,
// for example when checking uniqueness of an email.
func ErrFieldExists(field string) error { return errors.New(field + " already exists") }

// ErrFieldInvalid returns an error for an invalid form field. If the req
// variadic argument has at least one value, then that value is used as the
// requirement for that field.
func ErrFieldInvalid(field string, req ...string) error {
	msg := field + " is invalid"

	if len(req) > 0 {
		msg += ", " + req[0]
	}
	return errors.New(msg)
}

// ErrFieldRequired returns an error for a form field that is required.
func ErrFieldRequired(field string) error { return errors.New(field + " can't be blank") }

// Unmarshal decodes the given http.Request into the given Form interface, this
// supports decoding "application/json" requests too provided the Content-Type
// header is set accordingly.
func Unmarshal(f Form, r *http.Request) error {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return errors.Err(json.NewDecoder(r.Body).Decode(f))
	}

	if err := r.ParseForm(); err != nil {
		return errors.Err(err)
	}

	dec := schema.NewDecoder()
	dec.IgnoreUnknownKeys(true)
	return errors.Err(dec.Decode(f, r.Form))
}

// UnmarshalAndValidate parses the HTTP reques body and stores it in the given
// form, then validates it.
func UnmarshalAndValidate(f Form, r *http.Request) error {
	var unmarshalerr UnmarshalError

	if err := Unmarshal(f, r); err != nil {
		unmarshalerr, _ = errors.Cause(err).(UnmarshalError)
	}

	if err := f.Validate(); err != nil {
		if ferrs, ok := errors.Cause(err).(Errors); ok {
			if unmarshalerr.Field != "" && unmarshalerr.Err != nil {
				ferrs.Put(unmarshalerr.Field, unmarshalerr.Err)
			}
			return ferrs
		}
		return errors.Cause(err)
	}

	if unmarshalerr.Field != "" && unmarshalerr.Err != nil {
		ferrs := NewErrors()
		ferrs.Put(unmarshalerr.Field, unmarshalerr.Err)

		return ferrs
	}
	return nil
}
