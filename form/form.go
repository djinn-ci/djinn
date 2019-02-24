package form

import (
	"encoding/gob"
	"net/http"

	"github.com/andrewpillar/thrall/errors"

	"github.com/gorilla/schema"
)

type form map[string]string

type Form interface {
	Validate() error

	Get(key string) string
}

func init() {
	gob.Register(Errors(make(map[string][]string)))
	gob.Register(form(make(map[string]string)))
	gob.Register(Register{})
	gob.Register(Login{})
	gob.Register(Namespace{})
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

func Empty() form {
	return form(make(map[string]string))
}

func (f form) Get(key string) string {
	return f[key]
}

func (f form) Validate() error {
	return nil
}
