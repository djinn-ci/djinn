package form

import "bytes"

// Errors is a type for collecting any validation errors from a form. It is a
// map of string slices, where each key is the name of the field that failed
// validation, and each string in the slice is the error message.
type Errors map[string][]string

// NewErrors returns a new form.Errors type.
func NewErrors() Errors { return Errors(make(map[string][]string)) }

// Err returns the underlying error implementation if any errors exist in the
// map.
func (e Errors) Err() error {
	if len(e) == 0 {
		return nil
	}
	return e
}

// Error returns a formatted string of all the errors that occurred for each
// field. Each error message will be indented beneath the field name, for
// example,
//
//   username:
//       Username is required
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

// First returns the first error that is present for the given key. An empty
// string is returned if no error can be found.
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

// Put appends an error for the given key.
func (e *Errors) Put(key string, err error) {
	if err == nil {
		return
	}
	(*e)[key] = append((*e)[key], err.Error())
}
