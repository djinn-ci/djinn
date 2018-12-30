package driver

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"github.com/pkg/sftp"

	"golang.org/x/crypto/ssh"
)

type SSH struct {
	client *ssh.Client

	Address  string
	Username string
	Password string
	Timeout  time.Duration
}

func getHostKey(host string) (ssh.PublicKey, error) {
	f, err := os.Open(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))

	if err != nil {
		return nil, err
	}

	defer f.Close()

	s := bufio.NewScanner(f)

	var hostKey ssh.PublicKey

	for s.Scan() {
		fields := strings.Split(s.Text(), " ")

		if len(fields) != 3 {
			continue
		}

		if strings.Contains(fields[0], host) {
			var err error

			hostKey, _, _, _, err = ssh.ParseAuthorizedKey(s.Bytes())

			if err != nil {
				return nil, err
			}

			break
		}
	}

	if hostKey == nil {
		return nil, errors.New("no key for host " + host)
	}

	return hostKey, nil
}

func (d *SSH) Create(w io.Writer) error {
	fmt.Fprintf(w, "Running with SSH driver...\n")

	host, _, err := net.SplitHostPort(d.Address)

	if err != nil {
		return err
	}

	key, err := getHostKey(host)

	if err != nil {
		return err
	}

	cfg := &ssh.ClientConfig{
		User: d.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(d.Password),
		},
		Timeout:         d.Timeout,
		HostKeyCallback: ssh.FixedHostKey(key),
	}

	fmt.Fprintf(w, "Connecting to %s...\n", d.Address)

	cli, err := ssh.Dial("tcp", d.Address, cfg)

	if err != nil {
		return err
	}

	fmt.Fprintf(w, "Established SSH connection to %s...\n", d.Address)

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

	b, err := sess.CombinedOutput(buf.String())

	io.Copy(j.Buffer, bytes.NewBuffer(b))

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
