// Package qemu provides an implementation of a Driver driver for job execution.
// This driver uses the SSH driver to achieve job execution.
package qemu

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/driver"
	driverssh "djinn-ci.com/driver/ssh"
	"djinn-ci.com/errors"
	"djinn-ci.com/runner"

	"github.com/andrewpillar/fs"
)

// RealpathFunc is the function used for deriving the underlying path for an
// image to be used for booting the QEMU machine.
type RealpathFunc func(arch, image string) (string, error)

// Config is the struct used for initializing a new QEMU driver for build
// execution.
type Config struct {
	Arch   string // The architecture to use for virtualization.
	CPUs   int64  // The number of CPUs for the virtual machine.
	Memory int64  // The amount of memory in bytes for the virtual machine.
	Disks  string // The location to look for disk images.
	Image  string // The QCOW2 image to boot the virtual machine with.
}

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
	Realpath RealpathFunc
}

var (
	_ runner.Driver = (*Driver)(nil)
	_ driver.Config = (*Config)(nil)

	tcpMaxPort int64 = 65535

	archLookup = map[string]string{
		"amd64": "x86_64",
	}
)

// Init initializes a new driver for QEMU using the given io.Writer, and
// applying the given driver.Config.
func Init(w io.Writer, cfg driver.Config) runner.Driver {
	d := &Driver{
		port:   2222,
		Writer: w,
	}

	cfg.Apply(d)
	return d
}

// GetExpectedArch returns the QEMU arch that would be expected for the GOARCH.
func GetExpectedArch() (string, error) {
	arch, ok := archLookup[runtime.GOARCH]

	if !ok {
		return "", errors.New("qemu: unsupported architecture for qemu driver " + runtime.GOARCH)
	}
	return arch, nil
}

// MatchesGOARCH checks to see if the given QEMU arch matches the GOARCH. This
// is used by the worker to make sure that virtualization with KVM would be
// possible on the platform the worker is being run on.
func MatchesGOARCH(arch string) bool {
	return archLookup[arch] == runtime.GOARCH
}

func (cfg *Config) Merge(m map[string]string) driver.Config {
	cfg2 := (*cfg)
	cfg2.Image = m["image"]
	cfg2.Arch = m["arch"]

	return &cfg2
}

func (cfg *Config) Apply(d runner.Driver) {
	v, ok := d.(*Driver)

	if !ok {
		return
	}

	v.Arch = cfg.Arch
	v.CPUs = cfg.CPUs
	v.Memory = cfg.Memory
	v.Image = cfg.Image
	v.Realpath = func(arch, image string) (string, error) {
		path := filepath.Join(cfg.Disks, "qemu", arch, filepath.Join(strings.Split(image, "/")...))

		info, err := os.Stat(path)

		if err != nil {
			return "", err
		}
		if info.IsDir() {
			return "", errors.New("image is not a file")
		}
		return path, nil
	}
}

func (q *Driver) runCmd() error {
	disk, err := q.Realpath(q.Arch, q.Image)

	if err != nil {
		return err
	}

	var pidfile *os.File

	for q.port < tcpMaxPort {
		pidfile, err = os.CreateTemp("", "djinn-qemu-")

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
				fmt.Fprintf(q.Writer, "make sure your image exists in the namespace the build is being run from\n")
			} else {
				fmt.Fprintf(q.Writer, "failed to boot machine\n")
				fmt.Fprintf(q.Writer, buf.String()+"\n")
			}
			return err
		}
		break
	}

	q.pidfile = pidfile
	q.ssh = &driverssh.Driver{
		Writer:   io.Discard,
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
func (q *Driver) Create(c context.Context, env []string, pt runner.Passthrough, objects fs.FS) error {
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

	b, err := io.ReadAll(q.pidfile)

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

	// Wait for machine to boot before attempting to connect.
	time.Sleep(time.Second * 2)

	if err := q.ssh.Create(c, env, nil, objects); err != nil {
		return err
	}

	fmt.Fprintf(q.Writer, "Established SSH connection to machine as %s...\n\n", q.ssh.User)

	q.ssh.Writer = q.Writer
	err = q.ssh.PlaceObjects(pt, objects)
	q.ssh.Writer = io.Discard

	if err != nil {
		return errors.Err(err)
	}
	return nil
}

// Execute will perform the given job on the Driver machine via a call to the
// underlying SSH driver.
func (q *Driver) Execute(j *runner.Job, artifacts fs.FS) error { return q.ssh.Execute(j, artifacts) }

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
