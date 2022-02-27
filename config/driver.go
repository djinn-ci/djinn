package config

import (
	"io"

	"djinn-ci.com/driver"
	"djinn-ci.com/driver/docker"
	"djinn-ci.com/driver/os"
	"djinn-ci.com/driver/qemu"
	"djinn-ci.com/driver/ssh"
	"djinn-ci.com/errors"

	"github.com/andrewpillar/config"
)

type driverCfg struct {
	Driver struct {
		SSH    ssh.Config
		QEMU   qemu.Config
		Docker docker.Config
	} `config:",nogroup"`
}

var driverInits = map[string]driver.Init{
	"docker": docker.Init,
	"ssh":    ssh.Init,
	"os":     os.Init,
	"qemu":   qemu.Init,
}

func DecodeDriver(driverName, name string, r io.Reader) (driver.Init, driver.Config, error) {
	var cfg driverCfg

	dec := config.NewDecoder(name, decodeOpts...)

	if err := dec.Decode(&cfg, r); err != nil {
		return nil, nil, err
	}

	init, ok := driverInits[driverName]

	if !ok {
		return nil, nil, errors.New("unknown driver: " + driverName)
	}

	driverCfgs := map[string]driver.Config{
		"docker": &cfg.Driver.Docker,
		"ssh":    &cfg.Driver.SSH,
		"os":     os.Config{},
		"qemu":   &cfg.Driver.QEMU,
	}

	drivercfg, ok := driverCfgs[driverName]

	if !ok {
		return nil, nil, errors.New("unknown driver: " + driverName)
	}
	return init, drivercfg, nil
}
