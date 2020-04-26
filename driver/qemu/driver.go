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
	"time"

	driverssh "github.com/andrewpillar/thrall/driver/ssh"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"
)

type realpathFunc func(string, string) (string, error)

type QEMU struct {
	io.Writer

	ssh      *driverssh.SSH
	pidfile  *os.File
	process  *os.Process
	port     int64

	Arch     string
	CPUs     int64
	Memory   int64
	Key      string
	Image    string
	Realpath realpathFunc
}

var (
	_ runner.Driver = (*QEMU)(nil)

	tcpMaxPort int64 = 65535
)

func Init(w io.Writer, cfg map[string]interface{}) runner.Driver {
	cpus, ok := cfg["cpus"].(int64)

	if !ok {
		cpus = 1
	}

	memory, ok := cfg["memory"].(int64)

	if !ok {
		memory = 2048
	}

	key, ok := cfg["key"].(string)

	if !ok {
		key = filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
	}

	dir, ok := cfg["disks"].(string)

	if !ok {
		dir = "."
	}

	image, _ := cfg["image"].(string)

	return &QEMU{
		port:     2222,
		Writer:   w,
		Arch:     "x86_64",
		CPUs:     cpus,
		Memory:   memory,
		Key:      key,
		Image:    image,
		Realpath: func(arch, image string) (string, error) {
			path := filepath.Join(dir, arch, filepath.Join(strings.Split(image, "/")...))
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
}

func (q *QEMU) runCmd() error {
	disk, err := q.Realpath(q.Arch, q.Image)

	if err != nil {
		return err
	}

	for q.port < tcpMaxPort {
		hostfwd := net.JoinHostPort("127.0.0.1", strconv.FormatInt(q.port, 10))

		bin := fmt.Sprintf("qemu-system-%s", q.Arch)
		arg := []string{
			"-enable-kvm",
			"-daemonize",
			"-display",
			"none",
			"-pidfile",
			q.pidfile.Name(),
			"-smp",
			strconv.FormatInt(q.CPUs, 10),
			"-m",
			strconv.FormatInt(q.Memory, 10),
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

	q.ssh = &driverssh.SSH{
		Writer:  ioutil.Discard,
		Addr:    net.JoinHostPort("127.0.0.1", strconv.FormatInt(q.port, 10)),
		User:    "root",
		Key:     q.Key,
		Timeout: time.Duration(time.Second*60),
	}
	return nil
}

func (q *QEMU) Create(c context.Context, env []string, objs runner.Passthrough, p runner.Placer) error {
	var err error

	if q.Writer == nil {
		return errors.New("cannot create driver with nil io.Writer")
	}

	fmt.Fprintf(q.Writer, "Running with QEMU driver...\n")
	fmt.Fprintf(q.Writer, "Creating machine with arch %s...\n", q.Arch)

	q.pidfile, err = ioutil.TempFile("", "thrall-qemu-")

	if err != nil {
		return err
	}

	defer q.pidfile.Close()

	fmt.Fprintf(q.Writer, "Booting machine with image %s...\n", q.Image)

	if err := q.runCmd(); err != nil {
		return err
	}

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
