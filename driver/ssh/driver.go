// Package ssh provides an implemention of an Driver driver for executing jobs on.
package ssh

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/runner"

	"github.com/andrewpillar/fs"

	"github.com/pkg/sftp"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	Addr     string
	User     string
	Password string
	Timeout  time.Duration
}

// Driver provides an implementation of the runner.Driver interface for SSH
// connections.
type Driver struct {
	io.Writer

	client *ssh.Client
	env    []string

	Addr     string        // Addr is the full address (including port) of the server.
	User     string        // User is the user to use for SSH.
	Password string        // Password is the password to use for the SSH user.
	Timeout  time.Duration // Timeout is the duration for connection timeouts.
}

var (
	_ runner.Driver = (*Driver)(nil)
	_ driver.Config = (*Config)(nil)
)

func (cfg *Config) Merge(m map[string]string) driver.Config {
	cfg1 := (*cfg)
	cfg1.Addr = m["addr"]

	return &cfg1
}

func (cfg *Config) Apply(d runner.Driver) {
	v, ok := d.(*Driver)

	if !ok {
		return
	}

	v.User = cfg.User
	v.Password = cfg.Password
	v.Timeout = cfg.Timeout
}

// Init initializes a new driver for SSH using the given io.Writer, and
// applying the given driver.Config.
func Init(w io.Writer, cfg driver.Config) runner.Driver {
	d := &Driver{
		Writer: w,
	}

	cfg.Apply(d)
	return d
}

// Create opens up the Driver connection to the remote machine as configured via a
// previous call to Init. The given env slice is used to set an unexported
// variable for setting environment variables during job execution.
func (s *Driver) Create(c context.Context, env []string, pt runner.Passthrough, objects fs.FS) error {
	if s.Writer == nil {
		return errors.New("cannot create driver with nil io.Writer")
	}

	fmt.Fprintln(s.Writer, "Running with Driver ssh...")

	ticker := time.NewTicker(time.Second)
	after := time.After(s.Timeout)

	client := make(chan *ssh.Client)

	go func() {
		for {
			select {
			case <-ticker.C:
				cfg := &ssh.ClientConfig{
					User: s.User,
					Auth: []ssh.AuthMethod{
						ssh.Password(s.Password),
					},
					HostKeyCallback: ssh.InsecureIgnoreHostKey(),
					Timeout:         time.Second,
				}

				fmt.Fprintf(s.Writer, "Connecting to %s@%s...\n", s.User, s.Addr)

				cli, err := ssh.Dial("tcp", s.Addr, cfg)

				if err != nil {
					break
				}
				client <- cli
			}
		}
	}()

	select {
	case <-c.Done():
		return c.Err()
	case <-after:
		return fmt.Errorf("Timed out trying to connect to %s...\n", s.Addr)
	case cli := <-client:
		s.client = cli
	}

	fmt.Fprintf(s.Writer, "Established Driver connection to %s...\n\n", s.Addr)

	s.env = env
	return s.PlaceObjects(pt, objects)
}

// Execute will perform the given runner.Job. This turns it into a shell script
// that is executed on the remote machine once placed via SFTP. Before the job
// is executed however, the environment variables given via Create are set for
// the Driver session being used to invoke the script. The stderr, and stdout
// streams are forwarded to the underlying io.Writer.
func (s *Driver) Execute(j *runner.Job, artifacts fs.FS) error {
	sess, err := s.client.NewSession()

	if err != nil {
		return err
	}

	defer sess.Close()

	script := strings.Replace(j.Name+".sh", " ", "-", -1)
	buf := driver.CreateScript(j)

	cli, err := sftp.NewClient(s.client)

	if err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("EOF: make sure Subsystem in sshd_config on the image is configured correctly")
		}
		return err
	}

	defer cli.Close()

	f, err := cli.Create(script)

	if err != nil {
		return err
	}

	io.Copy(f, buf)

	if err := f.Chmod(0755); err != nil {
		return err
	}

	f.Close()

	for _, e := range s.env {
		parts := strings.SplitN(e, "=", 2)

		if len(parts) > 1 {
			if err := sess.Setenv(parts[0], parts[1]); err != nil {
				fmt.Fprintf(j.Writer, "Failed to setenv %s: %s\n", e, err)
			}
		}
	}

	sess.Stdout = j.Writer
	sess.Stderr = j.Writer

	if err := sess.Run("./" + script); err != nil {
		if _, ok := err.(*ssh.ExitError); !ok {
			return err
		}
	}

	if err := s.collectArtifacts(j, artifacts); err != nil {
		return err
	}

	cli.Remove(script)
	return nil
}

// Destroy closes the Driver connection.
func (s *Driver) Destroy() {
	if s.client != nil {
		s.client.Close()
	}
}

func (s *Driver) collectArtifact(w io.Writer, cli *sftp.Client, src, dst string, artifacts fs.FS) error {
	f, err := cli.Open(src)

	if err != nil {
		return err
	}

	defer f.Close()

	fmt.Fprintln(w, "Collecting artifact", src, "=>", dst)

	if _, err := artifacts.Put(fs.Rename(f, dst)); err != nil {
		return err
	}
	return nil
}

func (s *Driver) collectArtifacts(j *runner.Job, artifacts fs.FS) error {
	if len(j.Artifacts) == 0 {
		return nil
	}

	cli, err := sftp.NewClient(s.client)

	if err != nil {
		return err
	}

	defer cli.Close()

	fmt.Fprintln(j.Writer)

	for src, dst := range j.Artifacts {
		matches, err := cli.Glob(src)

		if err != nil {
			fmt.Fprintln(j.Writer, "artifact error:", err)
			continue
		}

		for _, path := range matches {
			dst := strings.Replace(dst, "*", filepath.Base(path), -1)

			if err := s.collectArtifact(j.Writer, cli, path, dst, artifacts); err != nil {
				fmt.Fprintln(j.Writer, "artifact error:", err)
			}
		}
	}
	return nil
}

// PlaceObjects copies the given objects from the given placer onto the
// environment via SFTP.
func (s *Driver) PlaceObjects(pt runner.Passthrough, objects fs.FS) error {
	if len(pt) == 0 {
		return nil
	}

	cli, err := sftp.NewClient(s.client)

	if err != nil {
		return err
	}

	defer cli.Close()

	for src, dst := range pt {
		func(src, dst string) {
			fmt.Fprintln(s.Writer, "Placing object", src, "=>", dst)

			f, err := cli.Create(dst)

			if err != nil {
				fmt.Fprintln(s.Writer, "object error:", errors.Cause(err))
				return
			}

			defer f.Close()

			if err := f.Chmod(0700); err != nil {
				fmt.Fprintln(s.Writer, "object error:", errors.Cause(err))
				return
			}

			object, err := objects.Open(src)

			if err != nil {
				fmt.Fprintln(s.Writer, "object error:", errors.Cause(err))
				return
			}

			defer object.Close()

			if _, err := io.Copy(f, object); err != nil {
				fmt.Fprintln(s.Writer, "object error:", errors.Cause(err))
			}
		}(src, dst)
	}

	fmt.Fprintln(s.Writer)
	return nil
}
