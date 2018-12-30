package driver

import (
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
	*SSH

	pidfile string
	process *os.Process

	Image   string
	Arch    string
	CPUs    string
	Memory  string
	HostFwd string
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
		return err
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
		"user,hostfwd=tcp:" + d.HostFwd + "-:22",
		"-drive",
		"file=" + d.Image + ",media=disk,snapshot=on,if=virtio",
	}

	fmt.Fprintf(w, "Booting machine with image %s...\n", filepath.Base(d.Image))

	cmd := exec.Command(bin, arg...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	buf := &bytes.Buffer{}

	_, err = io.Copy(buf, pidfile)

	if err != nil {
		return err
	}

	trim := strings.Trim(buf.String(), "\n")
	pid, err := strconv.ParseInt(trim, 10, 64)

	if err != nil {
		return err
	}

	d.process, err = os.FindProcess(int(pid))

	if err != nil {
		return err
	}

	if d.SSH == nil {
		return errors.New("SSH driver for QEMU not initialized")
	}

	after := time.After(d.SSH.Timeout)
	errs := make(chan error)
	done := make(chan bool)

	go func() {
		for {
			if err := d.SSH.Create(ioutil.Discard); err != nil {
				if strings.Contains(err.Error(), "unable to authenticate") {
					errs <- err
				}

				continue
			}

			done <- true
			break
		}
	}()

	select {
		case <-after:
			return errors.New("timed out waiting for SSH server to start")
		case err := <-errs:
			return err
		case <-done:
			fmt.Fprintf(w, "Established SSH connection to machine...\n\n")
	}

	return nil
}

func (d *QEMU) Execute(j *runner.Job, c runner.Collector) {
	d.SSH.Execute(j, c)
}

func (d *QEMU) Destroy() {
	if d.process != nil {
		d.process.Kill()
	}

	if d.pidfile != "" {
		os.Remove(d.pidfile)
	}
}
