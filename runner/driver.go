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

type DriverConf func(io.Writer) (Driver, error)

var (
	driverMu sync.RWMutex
	drivers  = make(map[string]DriverConf)
)

func ConfigureDriver(name string, fn DriverConf) {
	driverMu.Lock()
	defer driverMu.Unlock()
	drivers[name] = fn
}

func GetDriver(name string) (DriverConf, error) {
	driverMu.Lock()
	defer driverMu.Unlock()

	if fn, ok := drivers[name]; ok {
		return fn, nil
	}
	return nil, errors.New("no such driver:"+name)
}
