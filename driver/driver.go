// Package driver has implementations of the runner.Driver interface for the
// different drivers available in Djinn.
package driver

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/andrewpillar/djinn/database"
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

type Config map[string]string

var (
	_ sql.Scanner   = (*Config)(nil)
	_ driver.Valuer = (*Config)(nil)

	preamble = "#!/bin/sh\nexec 2>&1\nset -ex\n\n"
)

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

func (c Config) Value() (driver.Value, error) { return driver.Value(c.String()), nil }

func (c *Config) Scan(val interface{}) error {
	b, err := database.Scan(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b) == 0 {
		return nil
	}

	buf := bytes.NewBuffer(b)

	return errors.Err(json.NewDecoder(buf).Decode(c))
}

func (c *Config) String() string {
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(c)
	return buf.String()
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
