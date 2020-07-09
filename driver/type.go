package driver

import (
	"database/sql"
	"database/sql/driver"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/database"
)

// Type represents the type of driver being used.
type Type uint8

//go:generate stringer -type Type -linecomment
const (
	SSH Type = iota // ssh
	QEMU            // qemu
	Docker          // docker
)

var (
	_ sql.Scanner   = (*Type)(nil)
	_ driver.Valuer = (*Type)(nil)

	driversMap = map[string]Type{
		"ssh":    SSH,
		"qemu":   QEMU,
		"docker": Docker,
	}
)

// Scan assumes the given interface value is either a byte slice or a string,
// and turns the underlying value into the corresponding Type.
func (t *Type) Scan(val interface{}) error {
	b, err := database.Scan(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b) == 0 {
		(*t) = Type(0)
		return nil
	}
	return errors.Err(t.UnmarshalText(b))
}

// UnmarshalText unmarshals the given byte slice setting the underlying Type
// for that driver. If the byte slice is not of a valid driver then an error
// is returned.
func (t *Type) UnmarshalText(b []byte) error {
	var ok bool

	s := string(b)
	(*t), ok = driversMap[s]

	if !ok {
		return errors.New("unknown driver "+s)
	}
	return nil
}

// Value returns a string value of the driver Type to be inserted into the
// database.
func (t Type) Value() (driver.Value, error) { return driver.Value(t.String()), nil }
