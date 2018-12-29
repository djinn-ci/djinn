package runner

import "io"

type Driver interface {
	Create(w io.Writer) error

	Execute(j *Job, c Collector)

	Destroy()
}
