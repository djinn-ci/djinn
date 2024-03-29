package namespace

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"

	"djinn-ci.com/errors"
)

// Visibility represents the visibility level of a Namespace, there are three
// possible values it could be,
//
// Private  - only the owner and collaborators can view the Namespace and its
//
//	contents
//
// Internal - only authenticated users can view the Namespace and its contents
// Public   - anyone can view the Namespace and its contents, authenticated or
//
//	not
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
	i, err := driver.String.ConvertValue(val)

	if err != nil {
		return errors.Err(err)
	}

	s, ok := i.(string)

	if !ok {
		return errors.New("namespace: could not type assert Visibility to string")
	}

	if s == "" {
		(*v) = Visibility(0)
		return nil
	}

	if err := v.UnmarshalText([]byte(s)); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (v Visibility) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(v.String())

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

// UnmarshalJSON takes the given byte slice, and attempts to unmarshal it to a
// string from its JSON representation. This is then mapped to a known
// Visibility. This will return webutil.UnmarshalError if any error occurs.
func (v *Visibility) UnmarshalJSON(b []byte) error {
	var (
		s  string
		ok bool
	)

	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	(*v), ok = visMap[s]

	if !ok {
		return errors.New("unknown visibility " + s)
	}
	return nil
}

// UnmarshalText takes the given byte slice, and attempts to map it to a known
// Visibility. If it is a known Visibility, then that the current Visibility is
// set to that, otherwise webutil.UnmarshalError is returned.
func (v *Visibility) UnmarshalText(b []byte) error {
	var ok bool

	s := string(b)
	(*v), ok = visMap[s]

	if !ok {
		return errors.New("unknown visibility " + s)
	}
	return nil
}

// Value returns the string value of the current Visibility so it can be
// inserted into the database.
func (v Visibility) Value() (driver.Value, error) { return driver.Value(v.String()), nil }
