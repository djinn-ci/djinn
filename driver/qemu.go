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

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"
)

var (
	arches = []string{
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

	QemuDir string

	QemuCPUs   = int64(1)
	QemuMemory = int64(2048)
)

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

func (d *QEMU) Create(w io.Writer, objects []config.Passthrough) error {
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
		strconv.FormatInt(QemuCPUs, 10),
		"-m",
		strconv.FormatInt(QemuMemory, 10),
		"-net",
		"nic,model=virtio",
		"-net",
		"user,hostfwd=tcp:" + d.HostFwd + "-:22",
		"-drive",
		"file=" + filepath.Join(QemuDir, d.Image) + ",media=disk,snapshot=on,if=virtio",
	}

	fmt.Fprintf(w, "Booting machine with image %s...\n", d.Image)

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

	var attempts int

	for {
		if attempts == 5 {
			break
		}

		err = d.SSH.Create(ioutil.Discard, []config.Passthrough{})

		if err == nil {
			break
		}

		time.Sleep(time.Second * 5)

		attempts++
	}

	if err != nil {
		return err
	}

	fmt.Fprintf(w, "Established SSH connection to machine...\n")

	return d.placeObjects(w, objects)
}

func (d *QEMU) Execute(j *runner.Job, c runner.Collector) {
	d.SSH.Execute(j, c)
}

func (d *QEMU) Destroy() {
	d.SSH.Destroy()

	if d.process != nil {
		d.process.Kill()
	}

	if d.pidfile != "" {
		os.Remove(d.pidfile)
	}
}
