package driver

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"
)

var preamble = `#!/bin/sh

set -ex

`

func createScript(j *runner.Job) *bytes.Buffer {
	buf := bytes.NewBufferString(preamble)

	for _, cmd := range j.Commands {
		fmt.Fprintf(buf, "%s\n", cmd)
	}

	return buf
}

// Create a new driver from the given config, and from the specified
// environment variables. Driver environment variables are used to configure
// parts of the driver that will remain the same between each initialization
// of that driver. For example, SSH timeoutes, QEMU image locations, and QEMU
// CPUs and memory.
func NewEnv(w io.Writer, cfg config.Driver) (runner.Driver, error) {
	switch cfg.Type {
		case "docker":
			return &Docker{
				Writer:    w,
				mutex:     &sync.Mutex{},
				image:     cfg.Image,
				workspace: cfg.Workspace,
			}, nil
		case "qemu":
			hostfwd := os.Getenv("THRALL_QEMU_HOSTFWD")

			if hostfwd == "" {
				hostfwd = "127.0.0.1:2222"
			}

			timeout, err := strconv.ParseInt(os.Getenv("THRALL_SSH_TIMEOUT"), 10, 64)

			if err != nil {
				timeout = 10
			}

			dir := os.Getenv("THRALL_QEMU_DIR")

			if dir == "" {
				dir = "."
			}

			cpus, err := strconv.ParseInt(os.Getenv("THRALL_QEMU_CPUS"), 10, 64)

			if err != nil {
				cpus = 1
			}

			memory, err := strconv.ParseInt(os.Getenv("THRALL_QEMU_MEMORY"), 10, 64)

			if err != nil {
				memory = 2048
			}

			return &QEMU{
				Writer: w,
				SSH: &SSH{
					Writer:   ioutil.Discard,
					address:  hostfwd,
					username: os.Getenv("THRALL_SSH_USERNAME"),
					keyFile:  os.Getenv("THRALL_SSH_KEY"),
					timeout:  time.Duration(time.Second * time.Duration(timeout)),
				},
				dir:     dir,
				image:   cfg.Image,
				arch:    cfg.Arch,
				cpus:    int(cpus),
				memory:  int(memory),
				hostfwd: hostfwd,
			}, nil
		case "ssh":
			timeout, err := strconv.ParseInt(os.Getenv("THRALL_SSH_TIMEOUT"), 10, 64)

			if err != nil {
				timeout = 10
			}

			return &SSH{
				Writer:   w,
				address:  cfg.Address,
				username: os.Getenv("THRALL_SSH_USERNAME"),
				keyFile:  os.Getenv("THRALL_SSH_KEY"),
				timeout:  time.Duration(time.Second * time.Duration(timeout)),
			}, nil
		default:
			return nil, errors.New("unknown driver: " + cfg.Type)
	}
}
