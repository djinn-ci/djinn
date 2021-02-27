// Package qemu provides an implementation of a Driver driver for job execution.
// This driver uses the SSH driver to achieve job execution.
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

	driverssh "github.com/andrewpillar/djinn/driver/ssh"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/runner"
)

type realpathFunc func(string, string) (string, error)

// Driver provides an implementation of the runner.Driver interface for running
// jobs within a QEMU virtual machine. Under the hood this makes use of the
// ssh.Driver implementation to communicate with the virtual machine.
type Driver struct {
	io.Writer

	ssh     *driverssh.Driver
	pidfile *os.File
	process *os.Process
	port    int64

	Arch   string // Arch is the machine architecture that will be running the jobs.
	CPUs   int64  // CPUs specifies the number of CPUs to give the machine.
	Memory int64  // Memory specifies the amount of memory in bytes for the machine.
	Image  string // Image is the name of the QEMU image to use for the machine.

	// Realpath is a function callback that will return the full path of the
	// QCOW2 image to use when booting the Driver machine.
	Realpath realpathFunc
}

var (
	_ runner.Driver = (*Driver)(nil)

	tcpMaxPort int64 = 65535
)

// Init initializes a new Driver driver using the given io.Writer, and
// configuration map. Detailed below are the values, types, and default values
// that are used in the configuration map.
//
// CPUs - The number of CPUs to use for the Driver machine is specified via the
// "cpus" field in the map. This is expected to be an int64, by default it will
// be 1.
//
// Memory - The amount of memory to give the Driver machine is specified via the
// "memory" filed in the map. This is expected to be an int64, by default will
// be 2048.
//
// Disks - The directory to look in when looking up the location of the QCOW2
// images. This is expected to be a string, by default it will be ".".
//
// Image - The image to use when booting up the Driver machine, it is expected to
// be a string, there is no default value.
func Init(w io.Writer, cfg map[string]interface{}) runner.Driver {
	cpus, ok := cfg["cpus"].(int64)

	if !ok {
		cpus = 1
	}

	memory, ok := cfg["memory"].(int64)

	if !ok {
		memory = 2048
	}

	dir, ok := cfg["disks"].(string)

	if !ok {
		dir = "."
	}

	image, _ := cfg["image"].(string)

	return &Driver{
		port:   2222,
		Writer: w,
		Arch:   "x86_64",
		CPUs:   cpus,
		Memory: memory,
		Image:  image,
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

func (q *Driver) runCmd() error {
	disk, err := q.Realpath(q.Arch, q.Image)

	if err != nil {
		return err
	}

	var pidfile *os.File

	for q.port < tcpMaxPort {
		pidfile, err = ioutil.TempFile("", "djinn-qemu-")

		if err != nil {
			return err
		}

		if err != nil {
			return err
		}

		hostfwd := net.JoinHostPort("127.0.0.1", strconv.FormatInt(q.port, 10))

		bin := fmt.Sprintf("qemu-system-%s", q.Arch)
		arg := []string{
			"-enable-kvm",
			"-daemonize",
			"-display", "none",
			"-pidfile", pidfile.Name(),
			"-smp", strconv.FormatInt(q.CPUs, 10),
			"-m", strconv.FormatInt(q.Memory, 10),
			"-net", "nic,model=virtio",
			"-net", "user,hostfwd=tcp:" + hostfwd + "-:22",
			"-drive", "file=" + disk + ",media=disk,snapshot=on,if=virtio",
		}

		cmd := exec.Command(bin, arg...)

		var buf bytes.Buffer

		cmd.Stdin = os.Stdin
		cmd.Stdout = q.Writer
		cmd.Stderr = &buf

		if err := cmd.Run(); err != nil {
			pidfile.Close()
			os.Remove(pidfile.Name())

			if strings.Contains(buf.String(), "Could not set up host forwarding rule") {
				q.port++
				continue
			}

			if strings.Contains(buf.String(), "No such file or directory") {
				fmt.Fprintf(q.Writer, "failed to boot machine, couldn't find image %s\n", filepath.Base(disk))
			} else {
				fmt.Fprintf(q.Writer, "failed to boot machine\n")
			}
			return err
		}
		break
	}

	q.pidfile = pidfile
	q.ssh = &driverssh.Driver{
		Writer:   ioutil.Discard,
		Addr:     net.JoinHostPort("127.0.0.1", strconv.FormatInt(q.port, 10)),
		User:     "root",
		Password: "",
		Timeout:  time.Duration(time.Minute * 5),
	}
	return nil
}

// Create will boot a new Driver machine based on the configuration given via
// a previous call to Init. The Driver process will forward ports from the host
// to the guest to allow for SSH comms. The host port will be 2222, unless
// already taken in which case it will increment until all TCP ports have been
// exhausted.
func (q *Driver) Create(c context.Context, env []string, objs runner.Passthrough, p runner.Placer) error {
	var err error

	if q.Writer == nil {
		return errors.New("cannot create driver with nil io.Writer")
	}

	fmt.Fprintf(q.Writer, "Running with Driver qemu...\n")
	fmt.Fprintf(q.Writer, "Creating machine with arch %s...\n", q.Arch)

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

	fmt.Fprintf(q.Writer, "Established SSH connection to machine as %s...\n\n", q.ssh.User)

	q.ssh.Writer = q.Writer
	err = q.ssh.PlaceObjects(objs, p)
	q.ssh.Writer = ioutil.Discard

	return errors.Err(err)
}

// Execute will perform the given job on the Driver machine via a call to the
// underlying SSH driver.
func (q *Driver) Execute(j *runner.Job, c runner.Collector) { q.ssh.Execute(j, c) }

// Destroy will terminate the SSH connection to the Driver machine, and kill the
// underlying OS process, then will remove the PIDFILE for that process.
func (q *Driver) Destroy() {
	if q.ssh != nil {
		q.ssh.Destroy()
	}
	if q.process != nil {
		q.process.Kill()
	}
	if q.pidfile != nil {
		q.pidfile.Close()
		os.Remove(q.pidfile.Name())
	}
}
