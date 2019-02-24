package model

import (
	"database/sql/driver"

	"github.com/andrewpillar/thrall/errors"
)

type Visibility uint8

const (
	Private Visibility	= iota
	Internal
	Public
)

func (v *Visibility) Scan(val interface{}) error {
	if val == nil {
		(*v) = Visibility(0)
		return nil
	}

	sval, err := driver.String.ConvertValue(val)

	if err != nil {
		return errors.Err(err)
	}

	b, ok := sval.([]byte)

	if !ok {
		return errors.Err(errors.New("failed to scan Visibility"))
	}

	err = v.UnmarshalText(b)

	return errors.Err(err)
}

func (v *Visibility) UnmarshalText(b []byte) error {
	s := string(b)

	switch s {
		case "private":
			(*v) = Private
			return nil
		case "internal":
			(*v) = Internal
			return nil
		case "public":
			(*v) = Public
			return nil
		default:
			return errors.Err(errors.New("unknown visibility " + s))
	}
}

func (v Visibility) Value() (driver.Value, error) {
	switch v {
		case Private:
			return driver.Value("private"), nil
		case Internal:
			return driver.Value("internal"), nil
		case Public:
			return driver.Value("public"), nil
		default:
			return driver.Value(""), errors.Err(errors.New("unknown visibility level"))
	}
}
