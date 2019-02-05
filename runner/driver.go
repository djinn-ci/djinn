package runner

import (
	"io"

	"github.com/andrewpillar/thrall/config"
)

type Driver interface {
	Create(w io.Writer, env []string, objects []config.Passthrough, p Placer) error

	Execute(j *Job, c Collector)

	Destroy()
}
