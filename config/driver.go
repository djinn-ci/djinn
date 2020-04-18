package config

import (
	"io"

	"github.com/andrewpillar/thrall/errors"

	"github.com/pelletier/go-toml"
)

type Driver struct {
	Config map[string]string `toml:"-"`

	SSH  SSH
	Qemu Qemu
}

type SSH struct {
	User    string
	Key     string
	Timeout int
}

type Qemu struct {
	Disks  string
	Port   int64
	CPUs   int
	Memory int
}

func DecodeDriver(r io.Reader) (Driver, error) {
	dec := toml.NewDecoder(r)
	driver := Driver{}

	if err := dec.Decode(&driver); err != nil {
		return driver, errors.Err(err)
	}
	return driver, nil
}
