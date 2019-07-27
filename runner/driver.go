package runner

import (
	"context"
	"io"
)

type Driver interface {
	io.Writer

	Create(c context.Context, env []string, objects Passthrough, p Placer) error

	Execute(j *Job, c Collector)

	Destroy()
}
