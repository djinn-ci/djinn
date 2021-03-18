package config

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/andrewpillar/djinn/driver"
	"github.com/andrewpillar/djinn/driver/docker"
	"github.com/andrewpillar/djinn/driver/qemu"
	"github.com/andrewpillar/djinn/driver/ssh"
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

func DecodeDriver(name string, r io.Reader) (*driver.Registry, map[string]driver.Config, error) {
	errh := func(name string, line, col int, msg string) {
		fmt.Fprintf(os.Stderr, "%s,%d:%d - %s\n", name, line, col, msg)
	}

	p := newParser(name, r, errh)

	nodes := p.parse()

	if err := p.err(); err != nil {
		return nil, nil, err
	}

	var cfg0 driverCfg

	for _, n := range nodes {
		if err := cfg0.put(n); err != nil {
			return nil, nil, err
		}
	}

	drivers := driver.NewRegistry()

	cfgs := make(map[string]driver.Config)
	cfgs["docker"] = cfg0.docker
	cfgs["ssh"] = cfg0.ssh
	cfgs["qemu"] = cfg0.qemu

	return drivers, cfgs, nil
}

func (cfg *driverCfg) put(n *node) error {
	if n.name != "driver" {
		return n.err("unknown configuration parameter: " + n.name)
	}

	if n.label == "" {
		return n.err("unlabeled driver")
	}

	if n.body == nil {
		return n.err("driver must be a configuration block")
	}

	switch n.label {
	case "docker":
		var walkerr error

		n.body.walk(func(n *node) {
			switch n.name {
			case "host":
				if n.lit != stringLit {
					walkerr = n.err("docker host must be a string")
					return
				}
				cfg.docker.Host = n.value
			case "version":
				if n.lit != stringLit {
					walkerr = n.err("docker version must be a string")
					return
				}
				cfg.docker.Version = n.value
			default:
				walkerr = n.err("unknown ssh configuration parameter: " + n.name)
			}
		})
		return walkerr
	case "ssh":
		var walkerr error

		n.body.walk(func(n *node) {
			switch n.name {
			case "timeout":
				if n.lit != stringLit {
					walkerr = n.err("ssh timeout must be a valid duration string")
					return
				}

				cfg.ssh.Timeout, walkerr = time.ParseDuration(n.value)
			case "user":
				if n.lit != stringLit {
					walkerr = n.err("ssh user must be a string")
					return
				}
				cfg.ssh.User = n.value
			case "password":
				if n.lit != stringLit {
					walkerr = n.err("ssh password must be a string")
					return
				}
				cfg.ssh.Password = n.value
			default:
				walkerr = n.err("unknown ssh configuration parameter: " + n.name)
			}
		})
		return walkerr
	case "qemu":
		var walkerr error

		n.body.walk(func(n *node) {
			switch n.name {
			case "disks":
				if n.lit != stringLit {
					walkerr = n.err("qemu disks must be a string")
					return
				}
				cfg.qemu.Disks = n.value
			case "cpus":
				if n.lit != numberLit {
					walkerr = n.err("qemu cpus must be an integer")
					return
				}

				i, err := strconv.ParseInt(n.value, 10, 64)

				if err != nil {
					walkerr = n.err("qemu cpus must be an integer")
					return
				}
				cfg.qemu.CPUs = i
			case "memory":
				if n.lit != numberLit {
					walkerr = n.err("qemu memory must be an integer")
					return
				}

				i, err := strconv.ParseInt(n.value, 10, 64)

				if err != nil {
					walkerr = n.err("qemu memory must be an integer")
					return
				}
				cfg.qemu.Memory = i
			default:
				walkerr = n.err("unknown qemu configuration parameter: " + n.name)
			}
		})
		return walkerr
	default:
		return n.err("unknown driver configuration block: " + n.label)
	}
}
