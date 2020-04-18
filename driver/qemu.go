package driver

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"
)

type QEMU struct {
	io.Writer

	*SSH

	pidfile string
	process *os.Process

	dir     string
	image   string
	arch    string
	cpus    int
	memory  int
	hostfwd string
}

func resolveListenAddr(addr string) string {
	host, port, _ := net.SplitHostPort(addr)

	iport, _ := strconv.ParseInt(port, 10, 64)

	for {
		l, err := net.Listen("tcp", addr)

		if err != nil {
			iport++
			addr = net.JoinHostPort(host, strconv.FormatInt(iport, 10))
			continue
		}

		l.Close()
		break
	}

	return addr
}

func (d *QEMU) Create(c context.Context, env []string, objects runner.Passthrough, p runner.Placer) error {
	fmt.Fprintf(d.Writer, "Running with QEMU driver...\n")

	pidfile, err := ioutil.TempFile("", "thrall-qemu-")

	if err != nil {
		return err
	}

	out := make(chan string)

	// TODO: Need to actually check the error returned from qemu-system- binary
	// for bound ports to get the listen address. Concurrent runs of this
	// driver result in the same port trying to be bound to.
	go func() {
		out <- resolveListenAddr(d.hostfwd)
	}()

	select {
	case <-c.Done():
		return fmt.Errorf("Timed out trying to resolve host forward address...\n")
	case addr := <-out:
		d.hostfwd = addr
		d.SSH.address = addr
		break
	}

	d.pidfile = pidfile.Name()

	parts := strings.Split(d.image, "/")

	base := []string{d.dir, "_base", d.arch}
	disk := filepath.Join(append(base, parts...)...)

	info, err := os.Stat(disk)

	if err != nil || info.IsDir() {
		user := []string{d.dir}
		disk = filepath.Join(append(user, parts...)...)

		if _, err := os.Stat(disk); err != nil {
			return errors.New("could not find image")
		}
	}

	bin := fmt.Sprintf("qemu-system-%s", d.arch)
	arg := []string{
		"-enable-kvm",
		"-daemonize",
		"-display",
		"none",
		"-pidfile",
		d.pidfile,
		"-smp",
		fmt.Sprintf("%d", d.cpus),
		"-m",
		fmt.Sprintf("%d", d.memory),
		"-net",
		"nic,model=virtio",
		"-net",
		"user,hostfwd=tcp:" + d.hostfwd + "-:22",
		"-drive",
		"file=" + disk + ",media=disk,snapshot=on,if=virtio",
	}

	// Check if it's a custom image, we don't want to display the hash in the
	// build log.
	if strings.Contains(d.image, "::") {
		parts := strings.Split(d.image, "::")

		d.image = parts[0]
	}

	fmt.Fprintf(d.Writer, "Booting machine with image %s...\n", d.image)

	cmd := exec.Command(bin, arg...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = d.Writer
	cmd.Stderr = d.Writer

	if err := cmd.Run(); err != nil {
		return err
	}

	b, err := ioutil.ReadAll(pidfile)

	if err != nil {
		return err
	}

	trim := strings.Trim(string(b), "\n")
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

	if err := d.SSH.Create(c, env, runner.Passthrough{}, p); err != nil {
		return err
	}

	fmt.Fprintf(d.Writer, "Established SSH connection to machine...\n\n")

	d.SSH.Writer = d.Writer

	err = d.placeObjects(objects, p)

	d.SSH.Writer = ioutil.Discard

	return errors.Err(err)
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
