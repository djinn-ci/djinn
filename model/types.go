package model

import (
	"database/sql/driver"

	"github.com/andrewpillar/thrall/errors"
)

type Visibility uint8

type Status uint8

const (
	Private Visibility	= iota
	Internal
	Public

	Queued Status	= iota
	Running
	Passed
	Failed
)

func scanBytes(val interface{}) ([]byte, error) {
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

func (s *Status) Scan(val interface{}) error {
	b, err := scanBytes(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b) == 0 {
		(*s) = Status(0)
		return nil
	}

	err = s.UnmarshalText(b)

	return errors.Err(err)
}

func (s *Status) UnmarshalText(b []byte) error {
	str := string(b)

	switch str {
		case "queued":
			(*s) = Queued
			return nil
		case "running":
			(*s) = Running
			return nil
		case "passed":
			(*s) = Passed
			return nil
		case "failed":
			(*s) = Failed
			return nil
		default:
			return errors.Err(errors.New("unknown status " + str))
	}
}

func (s Status) Value() (driver.Value, error) {
	switch s {
		case Queued:
			return driver.Value("queued"), nil
		case Running:
			return driver.Value("running"), nil
		case Passed:
			return driver.Value("passed"), nil
		case Failed:
			return driver.Value("failed"), nil
		default:
			return driver.Value(""), errors.Err(errors.New("unknown status"))
	}
}

func (v *Visibility) Scan(val interface{}) error {
	b, err := scanBytes(val)

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
