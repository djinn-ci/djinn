package config

import (
	"io"
	"runtime"

	"github.com/andrewpillar/thrall/errors"

	"github.com/pelletier/go-toml"
)

type Worker struct {
	Parallelism int
	Drivers     []string
	Timeout     string
	Images      string

	Crypto struct {
		Key string
	}

	Redis struct {
		Addr     string
		Password string
	}

	Database struct {
		Addr     string
		Name     string
		Username string
		Password string
	}

	Artifacts string
	Objects   string

	SSH  SSH
	Qemu Qemu

	Log struct {
		Level string
		File  string
	}
}

func DecodeWorker(r io.Reader) (Worker, error) {
	dec := toml.NewDecoder(r)

	worker := Worker{}

	if err := dec.Decode(&worker); err != nil {
		return worker, errors.Err(err)
	}

	if worker.Parallelism == 0 {
		worker.Parallelism = runtime.NumCPU()
	}

	return worker, nil
}
