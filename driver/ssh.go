package driver

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/andrewpillar/thrall/runner"

	"github.com/pkg/sftp"

	"golang.org/x/crypto/ssh"
)

type SSH struct {
	client *ssh.Client

	Address  string
	Username string
	KeyFile  string
	Timeout  time.Duration
}

func (d *SSH) Create(w io.Writer) error {
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

	d.client = cli

	return nil
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

	sess.Stdout = j.Writer
	sess.Stderr = j.Writer

	err = sess.Run(buf.String())

	d.collectArtifacts(j, c)

	j.Success = err == nil

	if err != nil {
		_, ok := err.(*ssh.ExitError)

		if ok {
			err = nil
		}

		j.Failed(err)
	}
}

func (d *SSH) Destroy() {
	if d.client != nil {
		d.client.Close()
	}
}

func (d *SSH) collectArtifacts(j *runner.Job, c runner.Collector) {
	if len(j.Artifacts) == 0 {
		return
	}

	cli, err := sftp.NewClient(d.client)

	if err != nil {
		j.Failed(err)
		return
	}

	for _, art := range j.Artifacts {
		out := fmt.Sprintf("%s", filepath.Base(art))

		f, err := cli.Open(art)

		if err != nil {
			j.Failed(err)
			continue
		}

		defer f.Close()

		if err := c.Collect(out, f); err != nil {
			j.Failed(err)
		}
	}
}
