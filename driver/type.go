package driver

import (
	"database/sql"
	"database/sql/driver"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
)

type Type uint8

//go:generate stringer -type Type -linecomment
const (
	TypeSSH Type = iota // ssh
	TypeQEMU            // qemu
	TypeDocker          // docker
)

var (
	_ sql.Scanner   = (*Type)(nil)
	_ driver.Valuer = (*Type)(nil)

	driversMap = map[string]Type{
		"ssh":    TypeSSH,
		"qemu":   TypeQEMU,
		"docker": TypeDocker,
	}
)

func (t *Type) Scan(val interface{}) error {
	b, err := model.Scan(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b) == 0 {
		(*t) = Type(0)
		return nil
	}
	return errors.Err(t.UnmarshalText(b))
}

func (t *Type) UnmarshalText(b []byte) error {
	var ok bool

	s := string(b)
	(*t), ok = driversMap[s]

	if !ok {
		return errors.New("unknown driver "+s)
	}
	return nil
}

func (t Type) Value() (driver.Value, error) {
	return driver.Value(t.String()), nil
}
