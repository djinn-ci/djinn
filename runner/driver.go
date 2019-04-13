package runner

import (
	"io"

	"github.com/andrewpillar/thrall/config"
)

type Driver interface {
	io.Writer

	Create(env []string, objects []config.Passthrough, p Placer) error

	Execute(j *Job, c Collector)

	Destroy()
}
