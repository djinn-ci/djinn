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

	pidfile string
	process *os.Process
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
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password("secret"),
		},
		HostKeyCallback: ssh.FixedHostKey(key),
	}

	up := make(chan bool)
	errs := make(chan error)
	after := time.After(time.Duration(time.Second * 10))

	go func() {
		for {
			conn, err := ssh.Dial("tcp", "127.0.0.1:" + d.Port, cfg)

			if err != nil {
				if strings.Contains(err.Error(), "unable to authenticate") {
					errs <- err
					break
				}

				continue
			}

			conn.Close()

			up <- true
		}
	}()

	select {
		case <-after:
			return errors.New("timed out waiting for SSH server to start")
		case err := <-errs:
			return err
		case <-up:
	}

	return nil
}

func (d *QEMU) Execute(j *runner.Job, c runner.Collector) {

}

func (d *QEMU) Destroy() {
	if d.process != nil {
		d.process.Kill()
	}

	os.Remove(d.pidfile)
}
