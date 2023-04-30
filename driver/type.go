package driver

import (
	"database/sql"
	"database/sql/driver"

	"djinn-ci.com/errors"
)

// Type represents the type of driver being used.
type Type uint8

//go:generate stringer -type Type -linecomment
const (
	SSH    Type = iota // ssh
	QEMU               // qemu
	Docker             // docker
	OS                 // os
)

var (
	_ sql.Scanner   = (*Type)(nil)
	_ driver.Valuer = (*Type)(nil)

	driversMap = map[string]Type{
		"ssh":    SSH,
		"qemu":   QEMU,
		"docker": Docker,
		"os":     OS,
	}
)

// IsValid checks to see if the given driver type is valid.
func IsValid(typ string) bool {
	_, ok := driversMap[typ]
	return ok
}

func Lookup(typ string) (Type, error) {
	if typ, ok := driversMap[typ]; ok {
		return typ, nil
	}
	return 0, ErrUnknown(typ)
}

// Scan assumes the given interface value is either a byte slice or a string,
// and turns the underlying value into the corresponding Type.
func (t *Type) Scan(val interface{}) error {
	v, err := driver.String.ConvertValue(val)

	if err != nil {
		return errors.Err(err)
	}

	s, ok := v.(string)

	if !ok {
		return errors.New("driver: could not type assert Type to string")
	}

	if s == "" {
		(*t) = Type(0)
		return nil
	}

	if err := t.UnmarshalText([]byte(s)); err != nil {
		return errors.Err(err)
	}
	return nil
}

// UnmarshalText unmarshals the given byte slice setting the underlying Type
// for that driver. If the byte slice is not of a valid driver then an error
// is returned.
func (t *Type) UnmarshalText(b []byte) error {
	var ok bool

	s := string(b)
	(*t), ok = driversMap[s]

	if !ok {
		return errors.New("unknown driver " + s)
	}
	return nil
}

// Value returns a string value of the driver Type to be inserted into the
// database.
func (t Type) Value() (driver.Value, error) { return driver.Value(t.String()), nil }
