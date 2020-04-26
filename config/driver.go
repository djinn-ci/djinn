package config

import (
	"github.com/andrewpillar/thrall/errors"

	"github.com/pelletier/go-toml"
)

var (
	driverValidators = map[string]func(*toml.Tree)error{
		"ssh":    validateSSH,
		"docker": validateDocker,
		"qemu":   validateQEMU,
	}
)

func validateSSH(tree *toml.Tree) error {
	for _, key := range []string{"timeout", "key"} {
		if !tree.Has(key) {
			return errors.New("ssh config missing property "+key)
		}
	}

	if _, ok := tree.Get("timeout").(int64); !ok {
		return errors.New("ssh timeout is not an integer")
	}

	if _, ok := tree.Get("key").(string); !ok {
		return errors.New("ssh key is not a string")
	}
	return nil
}

func validateDocker(_ *toml.Tree) error { return nil }

func validateQEMU(tree *toml.Tree) error {
	for _, key := range []string{"key", "disks", "cpus", "memory"} {
		if !tree.Has(key) {
			return errors.New("qemu config missing property "+key)
		}
	}

	if _, ok := tree.Get("key").(string); !ok {
		return errors.New("qemu key is not a string")
	}

	if _, ok := tree.Get("disks").(string); !ok {
		return errors.New("qemu disks is not an string")
	}

	if _, ok := tree.Get("cpus").(int64); !ok {
		return errors.New("qemu cpus is not an integer")
	}

	if _, ok := tree.Get("memory").(int64); !ok {
		return errors.New("qemu memory is not an integer")
	}
	return nil
}

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
