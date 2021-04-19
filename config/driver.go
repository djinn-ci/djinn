package config

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"djinn-ci.com/driver"
	"djinn-ci.com/driver/docker"
	"djinn-ci.com/driver/qemu"
	"djinn-ci.com/driver/ssh"
)

type driverCfg struct {
	ssh    *ssh.Config
	qemu   *qemu.Config
	docker *docker.Config
}

var driverInits = map[string]driver.Init{
	"docker": docker.Init,
	"ssh":    ssh.Init,
	"qemu":   qemu.Init,
}

func dockerCfg(n *node) (*docker.Config, error) {
	var (
		cfg docker.Config
		err error
	)

	n.body.walk(func(n *node) {
		switch n.name {
		case "host":
			if n.lit != stringLit {
				err = n.err("docker host must be a string")
				return
			}
			cfg.Host = n.value
		case "version":
			if n.lit != stringLit {
				err = n.err("docker version must be a string")
				return
			}
			cfg.Version = n.value
		default:
			err = n.err("unknown ssh configuration parameter: " + n.name)
		}
	})
	return &cfg, err
}

func sshCfg(n *node) (*ssh.Config, error) {
	var (
		cfg ssh.Config
		err error
	)

	n.body.walk(func(n *node) {
		switch n.name {
		case "timeout":
			if n.lit != stringLit {
				err = n.err("ssh timeout must be a valid duration string")
				return
			}

			cfg.Timeout, err = time.ParseDuration(n.value)
		case "user":
			if n.lit != stringLit {
				err = n.err("ssh user must be a string")
				return
			}
			cfg.User = n.value
		case "password":
			if n.lit != stringLit {
				err = n.err("ssh password must be a string")
				return
			}
			cfg.Password = n.value
		default:
			err = n.err("unknown ssh configuration parameter: " + n.name)
		}
	})
	return &cfg, err
}

func qemuCfg(n *node) (*qemu.Config, error) {
	var (
		cfg qemu.Config
		err error
	)

	n.body.walk(func(n *node) {
		switch n.name {
		case "disks":
			if n.lit != stringLit {
				err = n.err("qemu disks must be a string")
				return
			}
			cfg.Disks = n.value
		case "cpus":
			if n.lit != numberLit {
				err = n.err("qemu cpus must be an integer")
				return
			}

			i, err := strconv.ParseInt(n.value, 10, 64)

			if err != nil {
				err = n.err("qemu cpus must be an integer")
				return
			}
			cfg.CPUs = i
		case "memory":
			if n.lit != numberLit {
				err = n.err("qemu memory must be an integer")
				return
			}

			i, err := strconv.ParseInt(n.value, 10, 64)

			if err != nil {
				err = n.err("qemu memory must be an integer")
				return
			}
			cfg.Memory = i
		default:
			err = n.err("unknown qemu configuration parameter: " + n.name)
		}
	})
	return &cfg, err
}

func DecodeDriver(dname, fname string, r io.Reader) (driver.Init, driver.Config, error) {
	errh := func(name string, line, col int, msg string) {
		fmt.Fprintf(os.Stderr, "%s,%d:%d - %s\n", name, line, col, msg)
	}

	p := newParser(fname, r, errh)

	nodes := p.parse()

	if err := p.err(); err != nil {
		return nil, nil, err
	}

	var (
		init driver.Init
		cfg  driver.Config
		err  error
	)

	for _, n := range nodes {
		if n.name != "driver" {
			return nil, nil, n.err("unknown configuration parameter: " + n.name)
		}
		if n.label == "" {
			return nil, nil, n.err("unlabeled driver")
		}
		if n.body == nil {
			return nil, nil, n.err("driver must be a configuration block")
		}

		var ok bool

		init, ok = driverInits[n.label]

		if !ok {
			return nil, nil, n.err("unknown driver: " + n.label)
		}

		switch n.label {
		case "docker":
			cfg, err = dockerCfg(n)
		case "ssh":
			cfg, err = sshCfg(n)
		case "qemu":
			cfg, err = qemuCfg(n)
		}
	}
	return init, cfg, err
}
