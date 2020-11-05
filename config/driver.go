package config

import (
	"io"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/driver"
	"github.com/andrewpillar/djinn/driver/docker"
	"github.com/andrewpillar/djinn/driver/ssh"
	"github.com/andrewpillar/djinn/driver/qemu"

	"github.com/pelletier/go-toml"
)

var driverInits = map[string]driver.Init{
	"docker": docker.Init,
	"ssh":    ssh.Init,
	"qemu":   qemu.Init,
}

func DecodeDriver(r io.Reader) (*driver.Registry, map[string]map[string]interface{}, error) {
	tree, err := toml.LoadReader(r)

	if err != nil {
		return nil, nil, errors.Err(err)
	}

	drivers := driver.NewRegistry()
	conf := make(map[string]map[string]interface{})

	for _, key := range tree.Keys() {
		subtree, ok := tree.Get(key).(*toml.Tree)

		if !ok {
			return nil, nil, errors.New("property is not a tree: " + key)
		}

		drivers.Register(key, driverInits[key])
		conf[key] = subtree.ToMap()
	}
	return drivers, conf, nil
}
