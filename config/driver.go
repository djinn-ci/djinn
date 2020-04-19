package config

import (
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/driver/docker"
	"github.com/andrewpillar/thrall/driver/ssh"
	"github.com/andrewpillar/thrall/driver/qemu"
	"github.com/andrewpillar/thrall/errors"

	"github.com/pelletier/go-toml"
)

var (
	driverValidators = map[string]driver.ValidatorFunc{
		"ssh":    ssh.Validate,
		"docker": docker.Validate,
		"qemu":   qemu.Validate,
	}

	driverConfigs = map[string]driver.ConfigureFunc{
		"ssh":    ssh.Configure,
		"docker": docker.Configure,
		"qemu":   qemu.Configure,
	}
)

func ValidateDrivers(tree *toml.Tree) error {
	keys := tree.Keys()

	if len(keys) == 0 {
		return errors.New("no drivers configured")
	}

	for _, key := range keys {
		if _, ok := driverValidators[key]; !ok {
			return errors.New("unknown driver configured: "+key)
		}

		subtree, ok := tree.Get(key).(*toml.Tree)

		if !ok {
			return errors.New("expected key-value configuration for driver: "+key)
		}

		if err := driverValidators[key](subtree); err != nil {
			return err
		}
	}
	return nil
}

func GetDriverConfig(name string) driver.ConfigureFunc {
	configure := driverConfigs[name]
	return configure
}
