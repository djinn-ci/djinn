// Package docker providers an implementation of a Driver driver for job
// execution. Each job executed will be done in a separate container, a volume
// is used to persist state across these containers.
package docker

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/runner"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// Config is the struct used for initializing a new Docker driver for build
// execution.
type Config struct {
	Host      string // The host running the Docker daemon.
	Version   string // The version of the Docker API.
	Image     string // The container image to use.
	Workspace string // The workspace in the container to mount the volume to.
}

// Driver provides an implementation of the runner.Dirver interface for running
// jobs within a Docker container.
type Driver struct {
	io.Writer

	client     *client.Client
	volume     types.Volume
	env        []string
	containers []string

	Host    string // Host is the host of the Docker registry to use.
	Version string // Version is the Docker API version to use.
	Image   string // Image is the name of the image to use in the Docker container.

	// Workspace specifies location on the Driver container to mount a volume
	// to so state can be persisted.
	Workspace string
}

var (
	_ runner.Driver = (*Driver)(nil)
	_ driver.Config = (*Config)(nil)
)

// Init initializes a new driver for Docker using the given io.Writer, and
// applying the given driver.Config.
func Init(w io.Writer, cfg driver.Config) runner.Driver {
	d := &Driver{
		Writer: w,
	}

	cfg.Apply(d)
	return d
}

func (cfg *Config) Apply(d runner.Driver) {
	v, ok := d.(*Driver)

	if !ok {
		return
	}

	v.Host = cfg.Host
	v.Version = cfg.Version
	v.Image = cfg.Image
	v.Workspace = cfg.Workspace
}

func (cfg *Config) Merge(m map[string]string) driver.Config {
	cfg1 := (*cfg)
	cfg1.Image = m["image"]
	cfg1.Workspace = m["workspace"]

	return &cfg1
}

// Create will create a volume, and pull down the configured image. The client
// to the Driver daemon is derived from the environment. Once the client has
// been established, the image volume is created, and the image is pulled down
// from the repository.
func (d *Driver) Create(c context.Context, env []string, objs runner.Passthrough, p runner.Placer) error {
	var err error

	if d.Writer == nil {
		return errors.New("cannot create driver with nil io.Writer")
	}

	fmt.Fprintf(d.Writer, "Running with Driver docker...\n")

	d.client, err = client.NewClientWithOpts(
		client.WithHost(d.Host),
		client.WithVersion(d.Version),
	)

	if err != nil {
		return err
	}

	done := make(chan struct{})
	errs := make(chan error)

	go func() {
		d.volume, err = d.client.VolumeCreate(c, volume.VolumeCreateBody{})

		if err != nil {
			errs <- err
			return
		}

		rc, err := d.client.ImagePull(c, d.Image, types.ImagePullOptions{})

		if err != nil {
			errs <- err
			return
		}

		defer rc.Close()

		io.Copy(ioutil.Discard, rc)
		done <- struct{}{}
	}()

	select {
	case <-c.Done():
		return c.Err()
	case <-done:
		break
	case err = <-errs:
		return err
	}

	image, _, err := d.client.ImageInspectWithRaw(c, d.Image)

	if err != nil {
		return err
	}

	fmt.Fprintf(d.Writer, "Using Driver image %s - %s...\n\n", d.Image, image.ID)

	d.env = env
	return d.placeObjects(objs, p)
}

// Execute performs the given runner.Job in a Driver container. Each job is
// turned into a shell script and placed onto an initial container. A
// subsequent container is then created, and the previously placed script is
// used as that new container's entrypoint. The logs for the container are
// forwarded to the underlying io.Writer.
func (d *Driver) Execute(j *runner.Job, c runner.Collector) {
	hostCfg := &container.HostConfig{
		Mounts: []mount.Mount{
			{Type: mount.TypeVolume, Source: d.volume.Name, Target: d.Workspace},
		},
	}
	cfg := &container.Config{
		Image: d.Image,
		Cmd:   []string{"true"},
	}

	ctx := context.Background()

	ctr, err := d.client.ContainerCreate(ctx, cfg, hostCfg, nil, "")

	if err != nil {
		j.Failed(err)
		return
	}

	d.containers = append(d.containers, ctr.ID)

	script := strings.Replace(j.Name+".sh", " ", "-", -1)
	buf := driver.CreateScript(j)

	header := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "/bin/" + script,
		Size:     int64(buf.Len()),
		Mode:     755,
	}

	pr, pw := io.Pipe()
	defer pr.Close()

	tw := tar.NewWriter(pw)

	go func() {
		defer tw.Close()
		defer pw.Close()

		tw.WriteHeader(header)
		io.Copy(tw, buf)
	}()

	err = d.client.CopyToContainer(ctx, ctr.ID, d.Workspace, pr, types.CopyToContainerOptions{})

	if err != nil {
		j.Failed(err)
		return
	}

	cfg.Cmd = []string{}
	cfg.Env = d.env
	cfg.Entrypoint = []string{script}

	ctr, err = d.client.ContainerCreate(ctx, cfg, hostCfg, nil, "")

	if err != nil {
		j.Failed(err)
		return
	}

	d.containers = append(d.containers, ctr.ID)

	if err := d.client.ContainerStart(ctx, ctr.ID, types.ContainerStartOptions{}); err != nil {
		j.Failed(err)
		return
	}

	logOpts := types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: false,
		Follow:     true,
	}

	go func() {
		rc, err := d.client.ContainerLogs(ctx, ctr.ID, logOpts)

		if err != nil {
			if err == io.EOF {
				return
			}
			return
		}

		defer rc.Close()
		stdcopy.StdCopy(j.Writer, ioutil.Discard, rc)
	}()

	status, errs := d.client.ContainerWait(ctx, ctr.ID, container.WaitConditionNotRunning)
	code := 0

	select {
	case err := <-errs:
		if err != nil {
			j.Failed(err)
			return
		}
	case resp := <-status:
		code = int(resp.StatusCode)
	}

	if code != 0 {
		j.Failed(nil)
	} else {
		j.Status = runner.Passed
	}

	if len(j.Artifacts.Values) > 0 {
		fmt.Fprintf(j.Writer, "\n")
	}

	for src, dst := range j.Artifacts.Values {
		src = d.Workspace + "/" + src

		fmt.Fprintf(j.Writer, "Collecting artifact %s => %s\n", src, dst)

		rc, _, err := d.client.CopyFromContainer(ctx, ctr.ID, src)

		if err != nil {
			fmt.Fprintf(
				j.Writer,
				"Failed to collect artifact %s => %s: %s\n",
				src,
				dst,
				errors.Cause(err),
			)
			continue
		}

		defer rc.Close()

		tr := tar.NewReader(rc)

		for {
			header, err := tr.Next()

			if err != nil {
				if err == io.EOF {
					break
				}
				fmt.Fprintf(j.Writer, "Failed to collect artifact %s => %s: %s\n", src, dst, err)
				break
			}

			switch header.Typeflag {
			case tar.TypeDir:
				break
			case tar.TypeReg:
				if _, err := c.Collect(dst, tr); err != nil {
					fmt.Fprintf(
						j.Writer,
						"Failed to collect artifact %s => %s: %s\n",
						src,
						dst,
						errors.Cause(err),
					)
				}
			}
		}
	}

	if j.Status == runner.Failed {
		j.Failed(nil)
	}
}

// Destroy will remove all containers created during job execution, and the
// volume. All of these operations are forced.
func (d *Driver) Destroy() {
	if d.client == nil {
		return
	}

	ctx := context.Background()

	opts := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	for _, ctr := range d.containers {
		d.client.ContainerRemove(ctx, ctr, opts)
	}
	d.client.VolumeRemove(ctx, d.volume.Name, true)
}

func (d *Driver) placeObjects(objs runner.Passthrough, p runner.Placer) error {
	if len(objs.Values) == 0 {
		return nil
	}

	cfg := &container.Config{
		Image: d.Image,
		Cmd:   []string{"true"},
	}

	hostCfg := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: d.volume.Name,
				Target: d.Workspace,
			},
		},
	}

	ctx := context.Background()

	ctr, err := d.client.ContainerCreate(ctx, cfg, hostCfg, nil, "")

	if err != nil {
		return err
	}

	for src, dst := range objs.Values {
		fmt.Fprintf(d.Writer, "Placing object %s => %s\n", src, dst)

		info, err := p.Stat(src)

		if err != nil {
			fmt.Fprintf(d.Writer, "Failed to place object %s => %s: %s\n", src, dst, errors.Cause(err))
			continue
		}

		header, err := tar.FileInfoHeader(info, info.Name())

		if err != nil {
			fmt.Fprintf(d.Writer, "Failed to place object %s => %s: %s\n", src, dst, errors.Cause(err))
			continue
		}

		header.Name = strings.TrimPrefix(dst, d.Workspace)

		pr, pw := io.Pipe()
		defer pr.Close()

		tw := tar.NewWriter(pw)

		go func(src string) {
			defer tw.Close()
			defer pw.Close()

			tw.WriteHeader(header)

			if _, err := p.Place(src, tw); err != nil {
				fmt.Fprintf(d.Writer, "Failed to place object %s => %s: %s\n", src, dst, errors.Cause(err))
			}
		}(src)

		err = d.client.CopyToContainer(ctx, ctr.ID, d.Workspace, pr, types.CopyToContainerOptions{})

		if err != nil {
			fmt.Fprintf(d.Writer, "Failed to place object %s => %s: %s\n", src, dst, errors.Cause(err))
		}
	}
	d.client.ContainerRemove(ctx, ctr.ID, types.ContainerRemoveOptions{})
	fmt.Fprintf(d.Writer, "\n")
	return nil
}
