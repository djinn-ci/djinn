package driver

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

type Docker struct {
	io.Writer

	client *client.Client
	volume types.Volume

	env        []string
	containers []string

	mutex *sync.Mutex

	image      string
	workspace  string
}

func NewDocker(image, workspace string) *Docker {
	return &Docker{
		mutex:      &sync.Mutex{},
		image:      image,
		workspace:  workspace,
	}
}

func (d *Docker) Create(env []string, objects runner.Passthrough, p runner.Placer) error {
	fmt.Fprintf(d.Writer, "Running with Docker driver...\n")

	cli, err := client.NewEnvClient()

	if err != nil {
		return err
	}

	d.client = cli

	ctx := context.Background()

	vol, err := d.client.VolumeCreate(ctx, volume.VolumeCreateBody{})

	if err != nil {
		return err
	}

	d.volume = vol

	fmt.Fprintf(d.Writer, "Pulling Docker image %s...\n", d.image)

	rc, err := d.client.ImagePull(ctx, d.image, types.ImagePullOptions{})

	if err != nil {
		return err
	}

	defer rc.Close()

	io.Copy(ioutil.Discard, rc)

	image, _, err := d.client.ImageInspectWithRaw(ctx, d.image)

	if err != nil {
		return err
	}

	fmt.Fprintf(d.Writer, "Using Docker image %s - %s...\n\n", d.image, image.ID)

	d.env = env

	return d.placeObjects(objects, p)
}

func (d *Docker) Execute(j *runner.Job, c runner.Collector) {
	buf := bytes.Buffer{}

	for i, cmd := range j.Commands {
		buf.WriteString("echo \"$ " + cmd + "\" && " + cmd)

		if i != len(j.Commands) - 1 {
			buf.WriteString(" && ")
		}
	}

	cfg := &container.Config{
		Image: d.image,
		Tty:   true,
		Env:   d.env,
		Cmd:   []string{"/bin/bash", "-c", buf.String()},
	}

	hostCfg := &container.HostConfig{
		Mounts: []mount.Mount{
			mount.Mount{
				Type:   mount.TypeVolume,
				Source: d.volume.Name,
				Target: d.workspace,
			},
		},
	}

	ctx := context.Background()

	ctr, err := d.client.ContainerCreate(ctx, cfg, hostCfg, nil, "")

	if err != nil {
		j.Failed(err)
		return
	}

	d.mutex.Lock()
	d.containers = append(d.containers, ctr.ID)
	d.mutex.Unlock()

	if err := d.client.ContainerStart(ctx, ctr.ID, types.ContainerStartOptions{}); err != nil {
		j.Failed(err)
		return
	}

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

	opts := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	}

	rc, err := d.client.ContainerLogs(ctx, ctr.ID, opts)

	if err != nil {
		j.Failed(err)
		return
	}

	defer rc.Close()

	io.Copy(j.Writer, rc)

	if len(j.Artifacts) > 0 {
		fmt.Fprintf(j.Writer, "\n")
	}

	for src, dst := range j.Artifacts {
		dst += ".tar"

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

		if err := c.Collect(dst, rc); err != nil {
			fmt.Fprintf(
				j.Writer,
				"Failed to collect artifact %s => %s: %s\n",
				src,
				dst,
				errors.Cause(err),
			)
		}
	}

	if j.Status == runner.Failed {
		j.Failed(nil)
	}
}

func (d *Docker) Destroy() {
	if d.client == nil {
		return
	}

	ctx := context.Background()

	for _, ctr := range d.containers {
		d.client.ContainerRemove(ctx, ctr, types.ContainerRemoveOptions{})
	}

	d.client.VolumeRemove(ctx, d.volume.Name, true)
}

func (d *Docker) placeObjects(objects runner.Passthrough, p runner.Placer) error {
	if len(objects) == 0 {
		return nil
	}

	cfg := &container.Config{
		Image: d.image,
		Tty:   true,
		Cmd:   []string{"true"},
	}

	hostCfg := &container.HostConfig{
		Mounts: []mount.Mount{
			mount.Mount{
				Type:   mount.TypeVolume,
				Source: d.volume.Name,
				Target: d.workspace,
			},
		},
	}

	ctx := context.Background()

	ctr, err := d.client.ContainerCreate(ctx, cfg, hostCfg, nil, "")

	if err != nil {
		return err
	}

	for src, dst := range objects {
		fmt.Fprintf(d.Writer, "Placing object %s => %s\n", src, dst)

		info, err := os.Stat(src)

		if err != nil {
			fmt.Fprintf(
				d.Writer,
				"Failed to place object %s => %s: %s\n",
				src,
				dst,
				errors.Cause(err),
			)
			continue
		}

		header, err := tar.FileInfoHeader(info, info.Name())

		if err != nil {
			fmt.Fprintf(
				d.Writer,
				"Failed to place object %s => %s: %s\n",
				src,
				dst,
				errors.Cause(err),
			)
			continue
		}

		pr, pw := io.Pipe()
		defer pr.Close()

		tw := tar.NewWriter(pw)

		go func(src string) {
			defer tw.Close()
			defer pw.Close()

			tw.WriteHeader(header)
			p.Place(src, tw)
		}(src)

		d.client.CopyToContainer(ctx, ctr.ID, d.workspace, pr, types.CopyToContainerOptions{})
	}

	d.client.ContainerRemove(ctx, ctr.ID, types.ContainerRemoveOptions{})

	fmt.Fprintf(d.Writer, "\n")

	return nil
}
