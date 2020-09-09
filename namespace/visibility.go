package namespace

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
)

// Visibility represents the visibility level of a Namespace, there are three
// possible values it could be,
//
// Private  - only the owner and collaborators can view the Namespace and its
//            contents
// Internal - only authenticated users can view the Namespace and its contents
// Public   - anyone can view the Namespace and its contents, authenticated or
//            not
type Visibility uint8

//go:generate stringer -type Visibility -linecomment
const (
	Private  Visibility = iota // private
	Internal                   // internal
	Public                     // public
)

var (
	_ sql.Scanner   = (*Visibility)(nil)
	_ driver.Valuer = (*Visibility)(nil)

	visMap = map[string]Visibility{
		"private":  Private,
		"internal": Internal,
		"public":   Public,
	}
)

// Scan scans the given interface value into a byte slice and will attempt to
// turn it into the correct Visibility value. If it succeeds then it set's it
// on the current Visibility, otherwise an error is returned.
func (v *Visibility) Scan(val interface{}) error {
	b, err := database.Scan(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b) == 0 {
		(*v) = Visibility(0)
		return nil
	}
	return errors.Err(v.UnmarshalText(b))
}

// UnmarshalJSON takes the given byte slice, and attempts to unmarshal it to a
// string from its JSON representation. This is then mapped to a known
// Visibility. This will return form.UnmarshalError if any error occurs.
func (v *Visibility) UnmarshalJSON(b []byte) error {
	var (
		s  string
		ok bool
	)

	if err := json.Unmarshal(b, &s); err != nil {
		return form.UnmarshalError{
			Field: "Visibility",
			Err:   err,
		}
	}

	(*v), ok = visMap[s]

	if !ok {
		return form.UnmarshalError{
			Field: "visibility",
			Err:   errors.New("unknown visibility " + s),
		}
	}
	return nil
}

// UnmarshalText takes the given byte slice, and attempts to map it to a known
// Visibility. If it is a known Visibility, then that the current Visibility is
// set to that, otherwise form.UnmarshalError is returned.
func (v *Visibility) UnmarshalText(b []byte) error {
	var ok bool

	s := string(b)
	(*v), ok = visMap[s]

	if !ok {
		return form.UnmarshalError{
			Field: "visibility",
			Err:   errors.New("unknown visibility " + s),
		}
	}
	return nil
}

// Value returns the string value of the current Visibility so it can be
// inserted into the database.
func (v Visibility) Value() (driver.Value, error) { return driver.Value(v.String()), nil }
