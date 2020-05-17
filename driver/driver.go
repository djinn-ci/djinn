// Package driver has implementations of the runner.Driver interface for the
// different drivers available in Thrall.
package driver

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"
)

// Init is the function for fully initializing a driver with the given
// io.Writer, and configuration passed in via the map.
type Init func(io.Writer, map[string]interface{}) runner.Driver

// Store is a struct that holds the different Init functions for initializing
// a driver.
type Store struct {
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

// NewStore returns a new Store for the driver Init functions.
func NewStore() *Store {
	return &Store{
		driversMU: sync.RWMutex{},
		drivers:   make(map[string]Init),
	}
}

// Register registers a driver Init function for the driver of the given name.
func (s *Store) Register(name string, fn Init) {
	s.driversMU.Lock()
	defer s.driversMU.Unlock()

	if _, ok := s.drivers[name]; ok {
		panic("driver " + name + " already registered")
	}
	s.drivers[name] = fn
}

// Get returns the driver Init function for the driver of the given name.
func (s *Store) Get(name string) (Init, error) {
	s.driversMU.Lock()
	defer s.driversMU.Unlock()

	if _, ok := s.drivers[name]; !ok {
		return nil, errors.New("unknown driver " +name)
	}
	return s.drivers[name], nil
}
