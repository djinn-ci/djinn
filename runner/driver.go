package runner

import (
	"context"
	"io"
	"sync"

	"github.com/andrewpillar/thrall/errors"
)

type Driver interface {
	io.Writer

	Create(c context.Context, env []string, objects Passthrough, p Placer) error

	Execute(j *Job, c Collector)

	Destroy()
}

type DriverConfFunc func(io.Writer) (Driver, error)

var (
	driverConfsMu sync.RWMutex
	driverConfs   = make(map[string]DriverConfFunc)
)

func RegisterDriver(name string, fn DriverConfFunc) {
	driverConfsMu.Lock()
	defer driverConfsMu.Unlock()
	driverConfs[name] = fn
}

func GetDriver(name string) (DriverConfFunc, error) {
	driverConfsMu.Lock()
	defer driverConfsMu.Unlock()

	if fn, ok := driverConfs[name]; ok {
		return fn, nil
	}
	return nil, errors.New("no such driver:"+name)
}
