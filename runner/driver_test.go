package runner

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/andrewpillar/fs"
)

type OS struct {
	io.Writer

	Chdir func() error
}

func (d OS) Create(_ context.Context, env []string, pt Passthrough, objects fs.FS) error {
	if d.Chdir != nil {
		if err := d.Chdir(); err != nil {
			return err
		}
	}

	for _, v := range env {
		parts := strings.SplitN(v, "=", 2)

		os.Setenv(parts[0], parts[1])
	}

	for src, dst := range pt {
		func(src, dst string) {
			f, err := os.Create(dst)

			if err != nil {
				fmt.Fprintln(d.Writer, "object error:", err)
				return
			}

			defer f.Close()

			object, err := objects.Open(src)

			if err != nil {
				fmt.Fprintln(d.Writer, "object error:", err)
				return
			}

			defer object.Close()

			if _, err := io.Copy(f, object); err != nil {
				fmt.Fprintln(d.Writer, "object error:", err)
				return

			}
		}(src, dst)
	}
	return nil
}

func (d OS) Execute(j *Job, artifacts fs.FS) error {
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

	j.status = Passed

	for src, dst := range j.Artifacts {
		func(src, dst string) {
			fmt.Fprintln(d.Writer, "Collecting artifact", src, "=>", dst)

			f, err := os.Open(src)

			if err != nil {
				fmt.Fprintln(d.Writer, "artifact error:", err)
				return
			}

			defer f.Close()

			artifact, err := fs.ReadFile(dst, f)

			if err != nil {
				fmt.Fprintln(d.Writer, "artifact error:", err)
				return
			}

			defer artifact.Close()

			if _, err := artifacts.Put(artifact); err != nil {
				fmt.Fprintln(d.Writer, "artifact error:", err)
			}
		}(src, dst)
	}
	return nil
}

func (d OS) Destroy() {}
