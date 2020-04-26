package driver

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"
)

type Init func(io.Writer, map[string]interface{}) runner.Driver

type Store struct {
	driversMU sync.RWMutex
	drivers   map[string]Init
}

var preamble = "#!/bin/sh\nexec 2>&1\nset -ex\n\n"

func CreateScript(j *runner.Job) *bytes.Buffer {
	buf := bytes.NewBufferString(preamble)

	for _, cmd := range j.Commands {
		fmt.Fprintf(buf, "%s\n", cmd)
	}
	return buf
}

func NewStore() *Store {
	return &Store{
		driversMU: sync.RWMutex{},
		drivers:   make(map[string]Init),
	}
}

func (s *Store) Register(name string, fn Init) {
	s.driversMU.Lock()
	defer s.driversMU.Unlock()

	if _, ok := s.drivers[name]; ok {
		panic("driver " + name + " already registered")
	}
	s.drivers[name] = fn
}

func (s *Store) Get(name string) (Init, error) {
	s.driversMU.Lock()
	defer s.driversMU.Unlock()

	if _, ok := s.drivers[name]; !ok {
		return nil, errors.New("unknown driver " +name)
	}
	return s.drivers[name], nil
}
