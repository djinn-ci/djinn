package form

import "encoding/gob"

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
