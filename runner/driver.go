package runner

import "io"

type Driver interface {
	io.Writer

	Create(env []string, objects Passthrough, p Placer) error

	Execute(j *Job, c Collector)

	Destroy()
}
