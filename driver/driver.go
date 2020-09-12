// Package driver has implementations of the runner.Driver interface for the
// different drivers available in Djinn.
package driver

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/runner"
)

// Init is the function for fully initializing a driver with the given
// io.Writer, and configuration passed in via the map.
type Init func(io.Writer, map[string]interface{}) runner.Driver

// Registry is a struct that holds the different Init functions for initializing
// a driver.
type Registry struct {
	driversMU sync.RWMutex
	drivers   map[string]Init
}

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

// NewRegistry returns a new Registry for the driver Init functions.
func NewRegistry() *Registry {
	return &Registry{
		driversMU: sync.RWMutex{},
		drivers:   make(map[string]Init),
	}
}

// Register registers a driver Init function for the driver of the given name.
func (r *Registry) Register(name string, fn Init) {
	r.driversMU.Lock()
	defer r.driversMU.Unlock()

	if _, ok := r.drivers[name]; ok {
		panic("driver " + name + " already registered")
	}
	r.drivers[name] = fn
}

// Get returns the driver Init function for the driver of the given name.
func (r *Registry) Get(name string) (Init, error) {
	r.driversMU.Lock()
	defer r.driversMU.Unlock()

	if _, ok := r.drivers[name]; !ok {
		return nil, errors.New("unknown driver " + name)
	}
	return r.drivers[name], nil
}
