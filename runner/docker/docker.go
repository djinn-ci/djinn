package docker

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
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
	client *client.Client
	volume types.Volume

	containers []string

	mutex *sync.Mutex

	image      string
	workspace  string
}

func New(image, workspace string) *Docker {
	return &Docker{
		mutex:      &sync.Mutex{},
		image:      image,
		workspace:  workspace,
	}
}

func (d *Docker) Create(w io.Writer) error {
	fmt.Fprintf(w, "Running with Docker driver...\n")

	cli, err := client.NewEnvClient()

	if err != nil {
		return errors.Err(err)
	}

	d.client = cli

	ctx := context.Background()

	vol, err := d.client.VolumeCreate(ctx, volume.VolumeCreateBody{})

	if err != nil {
		return errors.Err(err)
	}

	d.volume = vol

	fmt.Fprintf(w, "Pulling Docker image %s...\n", d.image)

	rc, err := d.client.ImagePull(ctx, d.image, types.ImagePullOptions{})

	if err != nil {
		return errors.Err(err)
	}

	defer rc.Close()

	io.Copy(ioutil.Discard, rc)

	image, _, err := d.client.ImageInspectWithRaw(ctx, d.image)

	if err != nil {
		return errors.Err(err)
	}

	fmt.Fprintf(w, "Using Docker image %s - %s...\n\n", d.image, image.ID)

	return nil
}

func (d *Docker) Execute(j *runner.Job) {
	cfg := &container.Config{
		Image: d.image,
		Tty:   true,
		Cmd:   []string{"/bin/bash", "-exc", strings.Join(j.Commands, ";")},
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
		j.Errors = append(j.Errors, err)
		j.Failed()
		return
	}

	d.containers = append(d.containers, ctr.ID)

	if err := d.client.ContainerStart(ctx, ctr.ID, types.ContainerStartOptions{}); err != nil {
		j.Errors = append(j.Errors, err)
		j.Failed()
		return
	}

	status, errs := d.client.ContainerWait(ctx, ctr.ID, container.WaitConditionNotRunning)
	code := 0

	select {
		case err := <-errs:
			if err != nil {
				j.Errors = append(j.Errors, err)
				j.Failed()
				return
			}
		case resp := <-status:
			code = int(resp.StatusCode)
	}

	opts := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	}

	rc, err := d.client.ContainerLogs(ctx, ctr.ID, opts)

	if err != nil {
		j.Errors = append(j.Errors, err)
		j.Failed()
		return
	}

	defer rc.Close()

	io.Copy(j.Buffer, rc)

	if code == 0 {
		j.Success = true
	} else {
		j.Failed()
	}
}

func (d *Docker) Destroy() {
	ctx := context.Background()

	for _, ctr := range d.containers {
		d.client.ContainerRemove(ctx, ctr, types.ContainerRemoveOptions{})
	}

	d.client.VolumeRemove(ctx, d.volume.Name, true)
}
