package model

import (
	"database/sql/driver"

	"github.com/andrewpillar/thrall/errors"
)

type Visibility uint8

type DriverType uint8

const (
	Private Visibility = iota
	Internal
	Public

	SSH DriverType = iota
	Qemu
	Docker
)

func scan(val interface{}) ([]byte, error) {
	if val == nil {
		return []byte{}, nil
	}

	str, err := driver.String.ConvertValue(val)

	if err != nil {
		return []byte{}, errors.Err(err)
	}

	b, ok := str.([]byte)

	if !ok {
		return []byte{}, errors.Err(errors.New("failed to scan bytes"))
	}

	return b, nil
}

func (t *DriverType) Scan(val interface{}) error {
	b, err := scan(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b) == 0 {
		(*t) = DriverType(0)
		return nil
	}

	return errors.Err(t.UnmarshalText(b))
}

func (t *DriverType) UnmarshalText(b []byte) error {
	str := string(b)

	switch str {
		case "ssh":
			(*t) = SSH
			return nil
		case "qemu":
			(*t) = Qemu
			return nil
		case "docker":
			(*t) = Docker
			return nil
		default:
			return errors.Err(errors.New("unknown driver " + str))
	}
}

func (t DriverType) Value() (driver.Value, error) {
	switch t {
		case SSH:
			return driver.Value("ssh"), nil
		case Qemu:
			return driver.Value("qemu"), nil
		case Docker:
			return driver.Value("docker"), nil
		default:
			return driver.Value(""), errors.Err(errors.New("unknown driver"))
	}
}

func (v *Visibility) Scan(val interface{}) error {
	b, err := scan(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b) == 0 {
		(*v) = Visibility(0)
		return nil
	}

	err = v.UnmarshalText(b)

	return errors.Err(err)
}

func (v *Visibility) UnmarshalText(b []byte) error {
	str := string(b)

	switch str {
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
			return errors.Err(errors.New("unknown visibility " + str))
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
