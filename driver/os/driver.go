// Package os provides a driver for executing commands directly on the OS.
// This should typically only be used as part of integration testing, or if you
// trust the builds being run.
package os

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/runner"

	"github.com/andrewpillar/fs"
)

type Driver struct {
	io.Writer

	curdir string

	Chdir func() error
}

// Tmpdir creates a temporary directory and changes into it for executing the OS
// driver.
func Tmpdir() error {
	dir, err := os.MkdirTemp("", "driver-os-*")

	if err != nil {
		return err
	}
	return os.Chdir(dir)
}

var _ runner.Driver = (*Driver)(nil)

func Init(w io.Writer, cfg driver.Config) runner.Driver {
	return &Driver{
		Writer: w,
		Chdir:  Tmpdir,
	}
}

type Config struct{}

func (cfg Config) Merge(_ map[string]string) driver.Config {
	return cfg
}

func (Config) Apply(_ runner.Driver) {}

func (d *Driver) Create(_ context.Context, env []string, pt runner.Passthrough, objects fs.FS) error {
	if d.Chdir != nil {
		dir, err := os.Getwd()

		if err != nil {
			return err
		}

		if err := d.Chdir(); err != nil {
			return err
		}
		d.curdir = dir
	}

	for _, v := range env {
		parts := strings.SplitN(v, "=", 2)

		os.Setenv(parts[0], parts[1])
	}

	for src, dst := range pt {
		func(src, dst string) {
			fmt.Fprintln(d.Writer, "Placing object", src, "=>", dst)

			f, err := os.Create(dst)

			if err != nil {
				fmt.Fprintln(d.Writer, "object error:", errors.Cause(err))
				return
			}

			defer f.Close()

			object, err := objects.Open(src)

			if err != nil {
				fmt.Fprintln(d.Writer, "object error:", errors.Cause(err))
				return
			}

			if _, err := io.Copy(f, object); err != nil {
				fmt.Fprintln(d.Writer, "object error:", errors.Cause(err))
			}
		}(src, dst)
	}
	return nil
}

func (d *Driver) Execute(j *runner.Job, artifacts fs.FS) error {
	for _, cmdline := range j.Commands {
		r := csv.NewReader(strings.NewReader(cmdline))
		r.Comma = ' '

		args, _ := r.Read()

		if args == nil || args[0] == "" {
			continue
		}

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = j.Writer
		cmd.Stderr = j.Writer

		if err := cmd.Run(); err != nil {
			return err
		}
	}

	for src, dst := range j.Artifacts {
		func(src, dst string) {
			fmt.Fprintln(d.Writer, "Collecting artifact", src, "=>", dst)

			f, err := os.Open(src)

			if err != nil {
				fmt.Fprintln(d.Writer, "artifact error:", errors.Cause(err))
				return
			}

			defer f.Close()

			artifact, err := fs.ReadFile(dst, f)

			if err != nil {
				fmt.Fprintln(d.Writer, "artifact error:", errors.Cause(err))
				return
			}

			defer artifact.Close()

			if _, err := artifacts.Put(artifact); err != nil {
				fmt.Fprintln(d.Writer, "artifact error:", errors.Cause(err))
			}
		}(src, dst)
	}
	return nil
}

func (d *Driver) Destroy() {
	if d.curdir != "" {
		os.Chdir(d.curdir)
	}
}
