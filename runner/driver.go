package runner

import (
	"io"

	"github.com/andrewpillar/thrall/config"
)

type Driver interface {
	Create(w io.Writer, objects []config.Passthrough) error

	Execute(j *Job, c Collector)

	Destroy()
}
