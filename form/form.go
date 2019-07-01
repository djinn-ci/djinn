package form

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"

	"github.com/gorilla/schema"
)

type Form interface {
	Validate() error

	Fields() map[string]string
}

func Unmarshal(f Form, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return errors.Err(err)
	}

	dec := schema.NewDecoder()
	dec.IgnoreUnknownKeys(true)

	return errors.Err(dec.Decode(f, r.Form))
}

func UnmarshalAndValidate(f Form, r *http.Request) error {
	if err := Unmarshal(f, r); err != nil {
		return errors.Err(err)
	}

	return f.Validate()
}
