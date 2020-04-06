package namespace

import (
	"database/sql"
	"database/sql/driver"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
)

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

func (v *Visibility) Scan(val interface{}) error {
	b, err := model.Scan(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b) == 0 {
		(*v) = Visibility(0)
		return nil
	}
	return errors.Err(v.UnmarshalText(b))
}

func (v *Visibility) UnmarshalText(b []byte) error {
	var ok bool

	s := string(b)
	(*v), ok = visMap[s]

	if !ok {
		return errors.New("unknown Visibility "+s)
	}
	return nil
}

func (v Visibility) Value() (driver.Value, error) {
	return driver.Value(v.String()), nil
}
