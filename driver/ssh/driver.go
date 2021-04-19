// Package ssh provides an implemention of an Driver driver for executing jobs on.
package ssh

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/runner"

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
	_ driver.Config  = (*Config)(nil)
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
func (s *Driver) Create(c context.Context, env []string, objs runner.Passthrough, p runner.Placer) error {
	if s.Writer == nil {
		return errors.New("cannot create driver with nil io.Writer")
	}

	fmt.Fprintf(s.Writer, "Running with Driver ssh...\n")

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
	return s.PlaceObjects(objs, p)
}

// Execute will perform the given runner.Job. This turns it into a shell script
// that is executed on the remote machine once placed via SFTP. Before the job
// is executed however, the environment variables given via Create are set for
// the Driver session being used to invoke the script. The stderr, and stdout
// streams are forwarded to the underlying io.Writer.
func (s *Driver) Execute(j *runner.Job, c runner.Collector) {
	sess, err := s.client.NewSession()

	if err != nil {
		j.Failed(err)
		return
	}

	defer sess.Close()

	script := strings.Replace(j.Name+".sh", " ", "-", -1)
	buf := driver.CreateScript(j)

	cli, err := sftp.NewClient(s.client)

	if err != nil {
		if err == io.EOF {
			err = errors.New("EOF: make sure Subsystem in sshd_config on the image is configured correctly")
		}
		j.Failed(err)
		return
	}

	defer cli.Close()

	f, err := cli.Create(script)

	if err != nil {
		j.Failed(err)
		return
	}

	io.Copy(f, buf)

	if err := f.Chmod(0755); err != nil {
		j.Failed(err)
		return
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
		if _, ok := err.(*ssh.ExitError); ok {
			err = nil
		}

		j.Failed(err)
	} else {
		j.Status = runner.Passed
	}

	s.collectArtifacts(j.Writer, j, c)
	cli.Remove(script)
}

// Destroy closes the Driver connection.
func (s *Driver) Destroy() {
	if s.client != nil {
		s.client.Close()
	}
}

func (s *Driver) collectArtifacts(w io.Writer, j *runner.Job, c runner.Collector) {
	if len(j.Artifacts.Values) == 0 {
		return
	}

	cli, err := sftp.NewClient(s.client)

	if err != nil {
		j.Failed(err)
		return
	}

	defer cli.Close()

	fmt.Fprintf(w, "\n")

	for src, dst := range j.Artifacts.Values {
		fmt.Fprintf(w, "Collecting artifact %s => %s\n", src, dst)

		f, err := cli.Open(src)

		if err != nil {
			fmt.Fprintf(
				w,
				"Failed to collect artifact %s => %s: %s\n",
				src,
				dst,
				errors.Cause(err),
			)
			continue
		}

		defer f.Close()

		if _, err := c.Collect(dst, f); err != nil {
			fmt.Fprintf(
				w,
				"Failed to collect artifact %s => %s: %s\n",
				src,
				dst,
				errors.Cause(err),
			)
		}
	}
}

// PlaceObjects copies the given objects from the given placer onto the
// environment via SFTP.
func (s *Driver) PlaceObjects(objects runner.Passthrough, p runner.Placer) error {
	if len(objects.Values) == 0 {
		return nil
	}

	cli, err := sftp.NewClient(s.client)

	if err != nil {
		return err
	}

	defer cli.Close()

	for src, dst := range objects.Values {
		fmt.Fprintf(s.Writer, "Placing object %s => %s\n", src, dst)

		f, err := cli.OpenFile(dst, os.O_WRONLY|os.O_APPEND|os.O_CREATE)

		if err != nil {
			fmt.Fprintf(
				s.Writer,
				"Failed to open object file on guest %s => %s: %s\n",
				src,
				dst,
				errors.Cause(err),
			)
			continue
		}

		defer f.Close()

		if err := f.Chmod(0700); err != nil {
			fmt.Fprintf(
				s.Writer,
				"Failed to chmod 0700 object %s => %s: %s\n",
				src,
				dst,
				errors.Cause(err),
			)
			continue
		}

		if _, err := p.Place(src, f); err != nil {
			fmt.Fprintf(
				s.Writer,
				"Failed to place object %s => %s: %s\n",
				src,
				dst,
				errors.Cause(err),
			)
			continue
		}
	}
	fmt.Fprintf(s.Writer, "\n")
	return nil
}
