package runner

import (
	"io"

	"github.com/andrewpillar/thrall/collector"
)

type Driver interface {
	Create(w io.Writer) error

	Execute(j *Job, c collector.Collector)

	Destroy()
}
