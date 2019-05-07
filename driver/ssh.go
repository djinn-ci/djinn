package driver

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"github.com/pkg/sftp"

	"golang.org/x/crypto/ssh"
)

type SSH struct {
	io.Writer

	client *ssh.Client

	env []string

	Address  string
	Username string
	KeyFile  string
	Timeout  time.Duration
}

func (d *SSH) Create(env []string, objects runner.Passthrough, p runner.Placer) error {
	fmt.Fprintf(d.Writer, "Running with SSH driver...\n")

	key, err := ioutil.ReadFile(d.KeyFile)

	if err != nil {
		return err
	}

	signer, err := ssh.ParsePrivateKey(key)

	if err != nil {
		return err
	}

	cfg := &ssh.ClientConfig{
		User: d.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		Timeout:         d.Timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	fmt.Fprintf(d.Writer, "Connecting to %s...\n", d.Address)

	cli, err := ssh.Dial("tcp", d.Address, cfg)

	if err != nil {
		return err
	}

	fmt.Fprintf(d.Writer, "Established SSH connection to %s...\n\n", d.Address)

	d.env = env
	d.client = cli

	return d.placeObjects(objects, p)
}

func (d *SSH) Execute(j *runner.Job, c runner.Collector) {
	sess, err := d.client.NewSession()

	if err != nil {
		j.Failed(err)
		return
	}

	defer sess.Close()

	script := j.Name + ".sh"
	buf := createScript(j)

	cli, err := sftp.NewClient(d.client)

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

	for _, e := range d.env {
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

	d.collectArtifacts(j.Writer, j, c)
}

func (d *SSH) Destroy() {
	if d.client != nil {
		d.client.Close()
	}
}

func (d *SSH) collectArtifacts(w io.Writer, j *runner.Job, c runner.Collector) {
	if len(j.Artifacts) == 0 {
		return
	}

	cli, err := sftp.NewClient(d.client)

	if err != nil {
		j.Failed(err)
		return
	}

	defer cli.Close()

	fmt.Fprintf(w, "\n")

	for src, dst := range j.Artifacts {
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

		if err := c.Collect(dst, f); err != nil {
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

func (d *SSH) placeObjects(objects runner.Passthrough, p runner.Placer) error {
	if len(objects) == 0 {
		return nil
	}

	cli, err := sftp.NewClient(d.client)

	if err != nil {
		return err
	}

	defer cli.Close()

	for src, dst := range objects {
		fmt.Fprintf(d.Writer, "Placing object %s => %s\n", src, dst)

		f, err := cli.Create(dst)

		if err != nil {
			fmt.Fprintf(
				d.Writer,
				"Failed to place object %s => %s: %s\n",
				src,
				dst,
				errors.Cause(err),
			)
			continue
		}

		defer f.Close()

		if err := p.Place(src, f); err != nil {
			fmt.Fprintf(
				d.Writer,
				"Failed to place object %s => %s: %s\n",
				src,
				dst,
				errors.Cause(err),
			)
			continue
		}
	}

	fmt.Fprintf(d.Writer, "\n")

	return nil
}
