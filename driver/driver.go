// Package driver has implementations of the runner.Driver interface for the
// different drivers available in Djinn.
package driver

import (
	"bytes"
	"fmt"
	"io"

	"djinn-ci.com/errors"
	"djinn-ci.com/runner"
)

type Config interface {
	// Apply applies the current configuration to the given Driver. This should
	// configure the Driver ready for build execution.
	Apply(d runner.Driver)

	// Merge in the given driver configuration from a build manifest, and return
	// a copy of the original config with the merged in values.
	Merge(m map[string]string) Config
}

type Error struct {
	Driver string
	Err    error
}

var (
	errUnknown  = errors.New("unknown driver")
	errDisabled = errors.New("driver disabled")
)

func ErrUnknown(name string) error {
	return &Error{
		Driver: name,
		Err:    errUnknown,
	}
}

func ErrDisabled(name string) error {
	return &Error{
		Driver: name,
		Err:    errDisabled,
	}
}

func (e *Error) Error() string {
	return "driver error: " + e.Driver + " - " + e.Err.Error()
}

// Init is the function for fully initializing a driver with the given
// io.Writer, and configuration passed in via the map.
type Init func(io.Writer, Config) runner.Driver

var preamble = "#!/bin/sh\nexec 2>&1\nset -ex\n\n"

// CreateScript returns a bytes.Buffer that contains a concatenation of the
// given runner.Job commands into a shell script. Each shell script is
// prepended with the given header,
//
//   #!/bin/sh
//   exec 2>&1
//   set -ex
func CreateScript(j *runner.Job) *bytes.Buffer {
	buf := bytes.NewBufferString(preamble)

	for _, cmd := range j.Commands {
		fmt.Fprintf(buf, "%s\n", cmd)
	}
	return buf
}
