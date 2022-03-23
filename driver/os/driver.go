// Package os provides a driver for executing commands directly on the OS.
// This should typically only be used as part of integration testing, or if you
// trust the builds being run.
package os

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/runner"
)

type Driver struct {
	io.Writer
}

var _ runner.Driver = (*Driver)(nil)

func Init(w io.Writer, cfg driver.Config) runner.Driver {
	return &Driver{
		Writer: w,
	}
}

type Config struct{}

func (cfg Config) Merge(_ map[string]string) driver.Config {
	return cfg
}

func (Config) Apply(_ runner.Driver) {}

func (d *Driver) logf(format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(d.Writer, format, a...)
}

func (d *Driver) Create(_ context.Context, env []string, objs runner.Passthrough, p runner.Placer) error {
	for _, v := range env {
		parts := strings.SplitN(v, "=", 2)

		os.Setenv(parts[0], parts[1])
	}

	for src, dst := range objs.Values {
		func(src, dst string) {
			f, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(0644))

			if err != nil {
				d.logf("Failed to open object file %s => %s: %s\n", src, dst, err)
				return
			}

			defer f.Close()

			if _, err := p.Place(src, f); err != nil {
				d.logf("Failed to place object %s => %s: %s\n", src, dst, errors.Cause(err))
			}
		}(src, dst)
	}
	return nil
}

func (d *Driver) Execute(j *runner.Job, c runner.Collector) {
	for _, cmdline := range j.Commands {
		args := strings.Split(cmdline, " ")

		if args[0] == "" {
			continue
		}

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = j.Writer
		cmd.Stderr = j.Writer

		if err := cmd.Run(); err != nil {
			j.Failed(err)
			return
		}
	}

	j.Status = runner.Passed

	for src, dst := range j.Artifacts.Values {
		d.logf("Collecting artifact %s => %s\n", src, dst)

		func(src, dst string) {
			f, err := os.Open(src)

			if err != nil {
				d.logf("Failed to collect artifact %s => %s: %s\n", src, dst, err)
				return
			}

			defer f.Close()

			if _, err := c.Collect(dst, f); err != nil {
				d.logf("Failed to collect artifact %s => %s: %s\n", src, dst, errors.Cause(err))
			}
		}(src, dst)
	}
}

func (*Driver) Destroy() {}
