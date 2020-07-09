package namespace

import (
	"database/sql"
	"database/sql/driver"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/database"
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
	Private Visibility = iota // private
	Internal                  // internal
	Public                    // public
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

// UnmarshalText takes the given byte slice, and attempts to map it to a known
// Visibility. If it is a known Visibility, then that the current Visibility is
// set to that, otherwise an error is returned.
func (v *Visibility) UnmarshalText(b []byte) error {
	var ok bool

	s := string(b)
	(*v), ok = visMap[s]

	if !ok {
		return errors.New("unknown Visibility "+s)
	}
	return nil
}

// Value returns the string value of the current Visibility so it can be
// inserted into the database.
func (v Visibility) Value() (driver.Value, error) { return driver.Value(v.String()), nil }
