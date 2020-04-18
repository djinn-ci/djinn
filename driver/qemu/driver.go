package qemu

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	driverssh "github.com/andrewpillar/thrall/driver/ssh"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"
)

type realpathFunc func(string, string) (string, error)

type Option func(*QEMU) (*QEMU, error)

type QEMU struct {
	io.Writer

	ssh      *driverssh.SSH
	sshopts  []driverssh.Option
	pidfile  *os.File
	process  *os.Process
	realpath realpathFunc
	image    string
	arch     string
	cpus     int
	memory   int
	port     int64
}

var (
	_ runner.Driver = (*QEMU)(nil)

	tcpMaxPort int64 = 65535
	timeout          = time.Duration(time.Second*60)
)

func errConf(err string) error { return errors.New("cannot configure QEMU driver:"+err) }

func Arch(arch string) Option {
	return func(q *QEMU) (*QEMU, error) {
		if arch == "" {
			return q, nil
		}
		q.arch = arch
		return q, nil
	}
}

func CPUs(cpus int) Option {
	return func(q *QEMU) (*QEMU, error) {
		if cpus == 0 {
			return q, nil
		}
		q.cpus = cpus
		return q, nil
	}
}

func Memory(memory int) Option {
	return func (q *QEMU) (*QEMU, error) {
		if memory == 0 {
			return q, nil
		}
		q.memory = memory
		return q, nil
	}
}

func Image(image string) Option {
	return func(q *QEMU) (*QEMU, error) {
		if image == "" {
			return q, errors.New("missing image for QEMU driver")
		}
		q.image = image
		return q, nil
	}
}

func Realpath(fn realpathFunc) Option {
	return func(q *QEMU) (*QEMU, error) {
		q.realpath = fn
		return q, nil
	}
}

func Key(key string) Option {
	return func(q *QEMU) (*QEMU, error) {
		if key == "" {
			return q, errors.New("missing key for QEMU driver")
		}
		q.sshopts = append(q.sshopts, driverssh.Key(key))
		return q, nil
	}
}

func Configure(opts ...Option) runner.DriverConf {
	return func(w io.Writer) (runner.Driver, error) {
		if w == nil {
			return nil, errConf("nil io.Writer")
		}

		var (
			qemu = &QEMU{
				arch:   "x86_64",
				cpus:   1,
				memory: 2048,
			}
			err error
		)

		for _, opt := range opts {
			qemu, err = opt(qemu)

			if err != nil {
				return nil, errConf(err.Error())
			}
		}
		return qemu, nil
	}
}

func (q *QEMU) runCmd() error {
	disk, err := q.realpath(q.arch, q.image)

	if err != nil {
		return err
	}

	for q.port < tcpMaxPort {
		hostfwd := net.JoinHostPort("127.0.0.1", strconv.FormatInt(q.port, 10))

		bin := fmt.Sprintf("qemu-system-%s", q.arch)
		arg := []string{
			"-enable-kvm",
			"-daemonize",
			"-display",
			"none",
			"-pidfile",
			q.pidfile.Name(),
			"-smp",
			fmt.Sprintf("%d", q.cpus),
			"-m",
			fmt.Sprintf("%d", q.memory),
			"-net",
			"nic,model=virtio",
			"-net",
			"user,hostfwd=tcp:" + hostfwd + "-:22",
			"-drive",
			"file=" + disk + ",media=disk,snapshot=on,if=virtio",
		}

		cmd := exec.Command(bin, arg...)

		buf := &bytes.Buffer{}

		cmd.Stdin = os.Stdin
		cmd.Stdout = q.Writer
		cmd.Stderr = io.MultiWriter(q.Writer, buf)

		if err := cmd.Run(); err != nil {
			if strings.Contains(buf.String(), "Could not set up host forwarding rule") {
				q.port++
				continue
			}
			return err
		}
		break
	}

	q.sshopts = append(
		q.sshopts,
		driverssh.Address(net.JoinHostPort("127.0.0.1", strconv.FormatInt(q.port, 10))),
	)
	return nil
}

func (q *QEMU) Create(c context.Context, env []string, objs runner.Passthrough, p runner.Placer) error {
	var err error

	fmt.Fprintf(q.Writer, "Running with QEMU driver...\n")

	q.pidfile, err = ioutil.TempFile("", "thrall-qemu-")

	if err != nil {
		return err
	}

	fmt.Fprintf(q.Writer, "Booting machine with image %s...\n", q.image)

	if err := q.runCmd(); err != nil {
		return err
	}

	ssh, err := driverssh.Configure(q.sshopts...)(q.Writer)

	if err != nil {
		return err
	}

	q.ssh = ssh.(*driverssh.SSH)

	b, err := ioutil.ReadAll(q.pidfile)

	if err != nil {
		return err
	}

	pid, err := strconv.ParseInt(strings.Trim(string(b), "\n"), 10, 64)

	if err != nil {
		return err
	}

	q.process, err = os.FindProcess(int(pid))

	if err != nil {
		return err
	}

	if err := q.ssh.Create(c, env, runner.Passthrough{}, p); err != nil {
		return err
	}

	fmt.Fprintf(q.Writer, "Etablished SSH connection to machine...\n\n")

	q.ssh.Writer = q.Writer
	err = q.ssh.PlaceObjects(objs, p)
	q.ssh.Writer = ioutil.Discard

	return errors.Err(err)
}

func (q *QEMU) Execute(j *runner.Job, c runner.Collector) { q.ssh.Execute(j, c) }

func (q *QEMU) Destroy() {
	q.ssh.Destroy()

	if q.process != nil {
		q.process.Kill()
	}
	if q.pidfile != nil {
		os.Remove(q.pidfile.Name())
	}
}
