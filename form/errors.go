package form

import "bytes"

type Errors map[string][]string

func NewErrors() Errors {
	return Errors(make(map[string][]string))
}

// Err return the final error if there are any
func (e Errors) Err() error {
	if len(e) == 0 {
		return nil
	}

	return e
}

func (e Errors) Error() string {
	buf := &bytes.Buffer{}

	for field, errs := range e {
		buf.WriteString(field + ":\n")

		for _, err := range errs {
			buf.WriteString("    " + err + "\n")
		}
	}

	return buf.String()
}

func (e Errors) First(key string) string {
	errs, ok := e[key]

	if !ok {
		return ""
	}

	if len(errs) == 0 {
		return ""
	}

	return errs[0]
}

func (e *Errors) Put(key string, err error) {
	if err == nil {
		return
	}

	(*e)[key] = append((*e)[key], err.Error())
}
