package driver

import (
	"fmt"
	"io"
	"os"
	"os/exec"

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
	Image  string
	Arch   string
	CPUs   string
	Memory string
	Port   string
}

func (d *QEMU) Create(w io.Writer) error {
	supported := false

	for _, arch := range arches {
		if arch == d.Arch {
			supported = true
			break
		}
	}

	if !supported {
		return errors.New("unsupported.Architecture: " + d.Arch)
	}

	bin := fmt.Sprintf("qemu-system-%s", d.Arch)
	arg := []string{
		"-daemonize",
		"-enable-kvm",
		"-display",
		"none",
		"-smp",
		d.CPUs,
		"-m",
		d.Memory,
		"-net",
		"nic,model=virtio",
		"-net",
		"user,hostfwd=tcp::" + d.Port + "-:22",
		"-drive",
		"file=" + d.Image + ",media=disk,snapshot=on,if=virtio",
	}

	cmd := exec.Command(bin, arg...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return errors.Err(cmd.Run())
}

func (d *QEMU) Execute(j *runner.Job, c runner.Collector) {

}

func (d *QEMU) Destroy() {

}
