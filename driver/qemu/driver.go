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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/driver"
	driverssh "github.com/andrewpillar/thrall/driver/ssh"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"github.com/pelletier/go-toml"
)

type realpathFunc func(string, string) (string, error)

type QEMU struct {
	io.Writer

	ssh      *driverssh.SSH
	sshtree  *toml.Tree
	sshopts  []driver.Option
	pidfile  *os.File
	process  *os.Process
	realpath realpathFunc
	image    string
	arch     string
	cpus     int64
	memory   int64
	port     int64
}

var (
	_ runner.Driver = (*QEMU)(nil)

	tcpMaxPort int64 = 65535
)

func Arch(arch string) driver.Option {
	return func(d runner.Driver) runner.Driver {
		if q, ok := d.(*QEMU); ok {
			q.arch = arch
			return q
		}
		return d
	}
}

func Image(image string) driver.Option {
	return func(d runner.Driver) runner.Driver {
		if q, ok := d.(*QEMU); ok {
			q.image = image
			return q
		}
		return d
	}
}

func Realpath(realpath realpathFunc) driver.Option {
	return func(d runner.Driver) runner.Driver {
		if q, ok := d.(*QEMU); ok {
			q.realpath = realpath
			return q
		}
		return d
	}
}

func Validate(tree *toml.Tree) error {
	for _, key := range []string{"key", "disks", "cpus", "memory"} {
		if !tree.Has(key) {
			return errors.New("qemu config missing property "+key)
		}
	}

	if _, ok := tree.Get("key").(string); !ok {
		return errors.New("qemu key is not a string")
	}

	if _, ok := tree.Get("disks").(string); !ok {
		return errors.New("qemu disks is not an string")
	}

	if _, ok := tree.Get("cpus").(int64); !ok {
		return errors.New("qemu cpus is not an integer")
	}

	if _, ok := tree.Get("memory").(int64); !ok {
		return errors.New("qemu memory is not an integer")
	}
	return nil
}

func Configure(w io.Writer, tree *toml.Tree, opts ...driver.Option) runner.Driver {
	cpus, ok := tree.Get("cpus").(int64)

	if !ok {
		cpus = 1
	}

	memory, ok := tree.Get("memory").(int64)

	if !ok {
		memory = 2048
	}

	disks, ok := tree.Get("disks").(string)

	if !ok {
		disks = "."
	}

	var qemu runner.Driver = &QEMU{
		Writer:   w,
		arch:     "x86_64",
		cpus:     cpus,
		memory:   memory,
		port:     2222,
		sshtree:  tree,
		realpath: func(arch, name string) (string, error) {
			name = filepath.Join(strings.Split(name, "/")...)
			path := filepath.Join(disks, arch, name)

			info, err := os.Stat(path)

			if err != nil {
				return "", err
			}
			if info.IsDir() {
				return "", errors.New("image is not a file")
			}
			return path, nil
		},
	}

	for _, opt := range opts {
		qemu = opt(qemu)
	}
	return qemu
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
			strconv.FormatInt(q.cpus, 10),
			"-m",
			strconv.FormatInt(q.memory, 10),
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
		cmd.Stderr = buf

		if err := cmd.Run(); err != nil {
			if strings.Contains(buf.String(), "Could not set up host forwarding rule") {
				q.port++
				continue
			}
			io.Copy(q.Writer, buf)
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

	if q.Writer == nil {
		return errors.New("cannot create driver with nil io.Writer")
	}

	fmt.Fprintf(q.Writer, "Running with QEMU driver...\n")
	fmt.Fprintf(q.Writer, "Creating machine with arch %s...\n", q.arch)

	q.pidfile, err = ioutil.TempFile("", "thrall-qemu-")

	if err != nil {
		return err
	}

	defer q.pidfile.Close()

	fmt.Fprintf(q.Writer, "Booting machine with image %s...\n", q.image)

	if err := q.runCmd(); err != nil {
		return err
	}

	q.ssh = driverssh.Configure(ioutil.Discard, q.sshtree, q.sshopts...).(*driverssh.SSH)

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
	if q.ssh != nil {
		q.ssh.Destroy()
	}
	if q.process != nil {
		q.process.Kill()
	}
	if q.pidfile != nil {
		os.Remove(q.pidfile.Name())
	}
}
