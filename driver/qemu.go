package driver

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"github.com/pkg/sftp"

	"golang.org/x/crypto/ssh"
)

var arches = []string{
	"aarch64",
	"alpha",
	"arm",
	"cris",
	"hppa",
	"i386",
	"lm32",
	"m68k",
	"microblaze",
	"microblazeel",
	"mips",
	"mips64",
	"mips64el",
	"mipsel",
	"moxie",
	"nios2",
	"or1k",
	"ppc",
	"ppc64",
	"ppcemb",
	"riscv32",
	"riscv64",
	"s390x",
	"sh4",
	"sh4eb",
	"sparc",
	"sparc64",
	"tricore",
	"unicore32",
	"x86_64",
	"xtensa",
	"xtensaeb",
}

type QEMU struct {
	Image  string
	Arch   string
	CPUs   string
	Memory string
	Port   string

	Username string
	Password string

	Timeout int64

	pidfile string
	process *os.Process

	client *ssh.Client
}

func getHostKey(host string) (ssh.PublicKey, error) {
	f, err := os.Open(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))

	if err != nil {
		return nil, errors.Err(err)
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
				return nil, errors.Err(err)
			}

			break
		}
	}

	if hostKey == nil {
		return nil, errors.Err(errors.New("no key for host " + host))
	}

	return hostKey, nil
}

func (d *QEMU) Create(w io.Writer) error {
	fmt.Fprintf(w, "Running with QEMU driver...\n")

	supported := false

	for _, arch := range arches {
		if arch == d.Arch {
			supported = true
			break
		}
	}

	if !supported {
		return errors.New("unsupported architecture: " + d.Arch)
	}

	pidfile, err := ioutil.TempFile("", "qemu-")

	if err != nil {
		return errors.Err(err)
	}

	d.pidfile = pidfile.Name()

	bin := fmt.Sprintf("qemu-system-%s", d.Arch)
	arg := []string{
		"-daemonize",
		"-enable-kvm",
		"-display",
		"none",
		"-pidfile",
		d.pidfile,
		"-smp",
		d.CPUs,
		"-m",
		d.Memory,
		"-net",
		"nic,model=virtio",
		"-net",
		"user,hostfwd=tcp:127.0.0.1:" + d.Port + "-:22",
		"-drive",
		"file=" + d.Image + ",media=disk,snapshot=on,if=virtio",
	}

	fmt.Fprintf(w, "Booting machine with image %s...\n", filepath.Base(d.Image))

	cmd := exec.Command(bin, arg...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := errors.Err(cmd.Run()); err != nil {
		return errors.Err(err)
	}

	buf := &bytes.Buffer{}

	_, err = io.Copy(buf, pidfile)

	if err != nil {
		return errors.Err(err)
	}

	trim := strings.Trim(buf.String(), "\n")
	pid, err := strconv.ParseInt(trim, 10, 64)

	if err != nil {
		return errors.Err(err)
	}

	d.process, err = os.FindProcess(int(pid))

	if err := errors.Err(err); err != nil {
		return errors.Err(err)
	}

	key, err := getHostKey("127.0.0.1")

	if err != nil {
		return errors.Err(err)
	}

	cfg := &ssh.ClientConfig{
		User: d.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(d.Password),
		},
		HostKeyCallback: ssh.FixedHostKey(key),
	}

	client := make(chan *ssh.Client)
	errs := make(chan error)
	after := time.After(time.Duration(time.Second * time.Duration(d.Timeout)))

	go func() {
		for {
			cli, err := ssh.Dial("tcp", "127.0.0.1:" + d.Port, cfg)

			if err != nil {
				if strings.Contains(err.Error(), "unable to authenticate") {
					errs <- err
					break
				}

				continue
			}

			client <- cli
		}
	}()

	select {
		case <-after:
			return errors.New("timed out waiting for SSH server to start")
		case err := <-errs:
			return err
		case d.client = <-client:
			fmt.Fprintf(w, "Established SSH connection to machine...\n\n")
	}

	return nil
}

func (d *QEMU) Execute(j *runner.Job, c runner.Collector) {
	sess, err := d.client.NewSession()

	if err != nil {
		j.Failed(err)
		return
	}

	defer sess.Close()

	buf := bytes.Buffer{}

	for i, cmd := range j.Commands {
		buf.WriteString("echo \"$ " + cmd + "\" && " + cmd)

		if i != len(j.Commands) - 1 {
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

func (d *QEMU) collectArtifacts(j *runner.Job, c runner.Collector) {
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

func (d *QEMU) Destroy() {
	if d.client != nil {
		d.client.Close()
	}

	if d.process != nil {
		d.process.Kill()
	}

	if d.pidfile != "" {
		os.Remove(d.pidfile)
	}
}
