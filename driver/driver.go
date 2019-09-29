package driver

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strconv"
	"time"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"
)

var (
	preamble = `#!/bin/sh
exec 2>&1

set -ex

`

	All = []string{
		"docker",
		"ssh",
		"qemu",
	}
)

func createScript(j *runner.Job) *bytes.Buffer {
	buf := bytes.NewBufferString(preamble)

	for _, cmd := range j.Commands {
		fmt.Fprintf(buf, "%s\n", cmd)
	}

	return buf
}

func New(w io.Writer, d config.Driver) (runner.Driver, error) {
	switch d.Config["type"] {
		case "docker":
			return &Docker{
				Writer:    w,
				image:     d.Config["image"],
				workspace: d.Config["workspace"],
			}, nil
		case "qemu":
			hostfwd := net.JoinHostPort("127.0.0.1", strconv.FormatInt(d.Qemu.Port, 10))

			return &QEMU{
				Writer: w,
				SSH: &SSH{
					Writer:   ioutil.Discard,
					address:  hostfwd,
					username: d.SSH.User,
					timeout:  time.Duration(time.Second * time.Duration(d.SSH.Timeout)),
					key:      d.SSH.Key,
				},
				dir:     d.Qemu.Disks,
				image:   d.Config["image"],
				arch:    "x86_64",
				cpus:    d.Qemu.CPUs,
				memory:  d.Qemu.Memory,
				hostfwd: hostfwd,
			}, nil
		case "ssh":
			return &SSH{
				Writer:   w,
				address:  d.Config["address"],
				username: d.SSH.User,
				timeout:  time.Duration(time.Second * time.Duration(d.SSH.Timeout)),
				key:      d.SSH.Key,
			}, nil
		default:
			return nil, errors.New("unknown driver: " + d.Config["type"])
	}
}
