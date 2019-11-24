package types

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"

	"github.com/andrewpillar/thrall/errors"
)

type Visibility uint8

type Driver uint8

type Trigger uint8

type TriggerData map[string]string

const (
	Private Visibility = iota
	Internal
	Public

	SSH Driver = iota
	Qemu
	Docker

	Manual Trigger = iota
	Push
	Pull
)

func Scan(val interface{}) ([]byte, error) {
	if val == nil {
		return []byte{}, nil
	}

	str, err := driver.String.ConvertValue(val)

	if err != nil {
		return []byte{}, errors.Err(err)
	}

	b, ok := str.([]byte)

	if !ok {
		return []byte{}, errors.Err(errors.New("failed to Scan bytes"))
	}

	return []byte(b), nil
}

func (t *Driver) Scan(val interface{}) error {
	b, err := Scan(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b) == 0 {
		(*t) = Driver(0)
		return nil
	}

	return errors.Err(t.UnmarshalText(b))
}

func (t *Driver) UnmarshalText(b []byte) error {
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

func (t Driver) String() string {
	switch t {
	case SSH:
		return "ssh"
	case Qemu:
		return "qemu"
	case Docker:
		return "docker"
	default:
		return ""
	}
}

func (t Driver) Value() (driver.Value, error) {
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

func (t *Trigger) Scan(val interface{}) error {
	b, err := Scan(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b) == 0 {
		(*t) = Trigger(0)
		return nil
	}

	return errors.Err(t.UnmarshalText(b))
}

func (t *Trigger) UnmarshalText(b []byte) error {
	str := string(b)

	switch str {
	case "manual":
		(*t) = Manual
		return nil
	case "push":
		(*t) = Push
		return nil
	case "pull":
		(*t) = Pull
		return nil
	default:
		return errors.Err(errors.New("unknown trigger " + str))
	}
}

func (t Trigger) Value() (driver.Value, error) {
	switch t {
	case Manual:
		return driver.Value("manual"), nil
	case Push:
		return driver.Value("push"), nil
	case Pull:
		return driver.Value("pull"), nil
	default:
		return driver.Value(""), errors.Err(errors.New("unknown trigger"))
	}
}

func (t TriggerData) Scan(val interface{}) error {
	b, err := Scan(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b) == 0 {
		return nil
	}

	buf := bytes.NewBuffer(b)
	dec := json.NewDecoder(buf)

	return errors.Err(dec.Decode(&t))
}

func (t *TriggerData) Set(key, val string) {
	if (*t) == nil {
		(*t) = make(map[string]string)
	}

	(*t)[key] = val
}

func (t TriggerData) Value() (driver.Value, error) {
	buf := &bytes.Buffer{}

	enc := json.NewEncoder(buf)
	enc.Encode(&t)

	return driver.Value(buf.String()), nil
}

func (v *Visibility) Scan(val interface{}) error {
	b, err := Scan(val)

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
