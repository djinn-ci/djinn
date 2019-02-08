package driver

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"github.com/pkg/sftp"

	"golang.org/x/crypto/ssh"
)

type SSH struct {
	client *ssh.Client

	env []string

	Address  string
	Username string
	KeyFile  string
	Timeout  time.Duration
}

func (d *SSH) Create(w io.Writer, env []string, objects []config.Passthrough, p runner.Placer) error {
	fmt.Fprintf(w, "Running with SSH driver...\n")

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

	fmt.Fprintf(w, "Connecting to %s...\n", d.Address)

	cli, err := ssh.Dial("tcp", d.Address, cfg)

	if err != nil {
		return err
	}

	fmt.Fprintf(w, "Established SSH connection to %s...\n\n", d.Address)

	d.env = env
	d.client = cli

	return d.placeObjects(w, objects, p)
}

func (d *SSH) Execute(j *runner.Job, c runner.Collector) {
	sess, err := d.client.NewSession()

	if err != nil {
		j.Failed(err)
		return
	}

	defer sess.Close()

	buf := bytes.Buffer{}
	l := len(j.Commands) - 1

	for i, cmd := range j.Commands {
		buf.WriteString(`echo "$ ` + cmd + ` " && ` + cmd)

		if i != l {
			buf.WriteString(" && ")
		}
	}

	for _, e := range d.env {
		parts := strings.Split(e, "=")

		if len(parts) > 1 {
			if err := sess.Setenv(parts[0], parts[1]); err != nil {
				fmt.Fprintf(j.Writer, "Failed to setenv %s: %s\n", e, err)
			}
		}
	}

	sess.Stdout = j.Writer
	sess.Stderr = j.Writer

	err = sess.Run(buf.String())

	if err != nil {
		if _, ok := err.(*ssh.ExitError); ok {
			err = nil
		}

		j.Failed(err)
	} else {
		j.Success = true
	}

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

	for _, a := range j.Artifacts {
		fmt.Fprintf(w, "Collecting artifact %s => %s\n", a.Source, a.Destination)

		f, err := cli.Open(a.Source)

		if err != nil {
			fmt.Fprintf(w, "Failed to collect artifact %s => %s: %s\n", a.Source, a.Destination, errors.Cause(err))
			continue
		}

		defer f.Close()

		if err := c.Collect(a.Destination, f); err != nil {
			fmt.Fprintf(w, "Failed to collect artifact %s => %s: %s\n", a.Source, a.Destination, errors.Cause(err))
		}
	}
}

func (d *SSH) placeObjects(w io.Writer, objects []config.Passthrough, p runner.Placer) error {
	if len(objects) == 0 {
		return nil
	}

	cli, err := sftp.NewClient(d.client)

	if err != nil {
		return err
	}

	defer cli.Close()

	for _, o := range objects {
		fmt.Fprintf(w, "Placing object %s => %s\n", o.Source, o.Destination)

		f, err := cli.Create(o.Destination)

		if err != nil {
			fmt.Fprintf(w, "Failed to place object %s => %s: %s\n", o.Source, o.Destination, errors.Cause(err))
			continue
		}

		defer f.Close()

		if err := p.Place(o.Source, f); err != nil {
			fmt.Fprintf(w, "Failed to place object %s => %s: %s\n", o.Source, o.Destination, errors.Cause(err))
			continue
		}
	}

	fmt.Fprintf(w, "\n")

	return nil
}
