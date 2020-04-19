package ssh

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"github.com/pelletier/go-toml"

	"github.com/pkg/sftp"

	"golang.org/x/crypto/ssh"
)

type Option func(*SSH) (*SSH, error)

type SSH struct {
	io.Writer

	client  *ssh.Client
	env     []string
	addr    string
	user    string
	key     string
	timeout time.Duration
}

var _ runner.Driver = (*SSH)(nil)

func Address(addr string) driver.Option {
	return func(d runner.Driver) runner.Driver {
		if s, ok := d.(*SSH); ok {
			s.addr = addr
			return s
		}
		return d
	}
}

func Validate(tree *toml.Tree) error {
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

func Configure(w io.Writer, tree *toml.Tree, opts ...driver.Option) runner.Driver {
	timeout, ok := tree.Get("timeout").(int)

	if !ok {
		timeout = 60
	}

	var ssh runner.Driver = &SSH{
		Writer: w,
		user:   "root",
		key:     tree.Get("key").(string),
		timeout: time.Duration(time.Second*time.Duration(timeout)),
	}

	for _, opt := range opts {
		ssh = opt(ssh)
	}
	return ssh
}

func (s *SSH) Create(c context.Context, env []string, objs runner.Passthrough, p runner.Placer) error {
	if s.Writer == nil {
		return errors.New("cannot create driver with nil io.Writer")
	}

	fmt.Fprintf(s.Writer, "Running with SSH driver...\n")

	ticker := time.NewTicker(time.Second)
	after := time.After(s.timeout)

	client := make(chan *ssh.Client)

	b, err := ioutil.ReadFile(s.key)

	if err != nil {
		return err
	}

	signer, err := ssh.ParsePrivateKey(b)

	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ticker.C:
				cfg := &ssh.ClientConfig{
					User: "root",
					Auth: []ssh.AuthMethod{
						ssh.PublicKeys(signer),
					},
					HostKeyCallback: ssh.InsecureIgnoreHostKey(),
					Timeout:         time.Second,
				}

				fmt.Fprintf(s.Writer, "Connecting to %s...\n", s.addr)

				cli, err := ssh.Dial("tcp", s.addr, cfg)

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
		return fmt.Errorf("Timed out trying to connect to %s...\n", s.addr)
	case cli := <-client:
		s.client = cli
	}

	fmt.Fprintf(s.Writer, "Established SSH connection to %s...\n\n", s.addr)

	s.env = env
	return s.PlaceObjects(objs, p)
}

func (s *SSH) Execute(j *runner.Job, c runner.Collector) {
	sess, err := s.client.NewSession()

	if err != nil {
		j.Failed(err)
		return
	}

	defer sess.Close()

	script := strings.Replace(j.Name + ".sh", " ", "-", -1)
	buf := driver.CreateScript(j)

	cli, err := sftp.NewClient(s.client)

	if err != nil {
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

	cli.Remove(script)
	s.collectArtifacts(j.Writer, j, c)
}

func (s *SSH) Destroy() {
	if s.client != nil {
		s.client.Close()
	}
}

func (s *SSH) collectArtifacts(w io.Writer, j *runner.Job, c runner.Collector) {
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

func (s *SSH) PlaceObjects(objects runner.Passthrough, p runner.Placer) error {
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
				"Failed to place object %s => %s: %s\n",
				src,
				dst,
				errors.Cause(err),
			)
			continue
		}

		defer f.Close()

		if err := f.Chmod(0600); err != nil {
			fmt.Fprintf(
				s.Writer,
				"Failed to place object %s => %s: %s\n",
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
