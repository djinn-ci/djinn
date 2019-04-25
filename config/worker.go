package config

import (
	"io"
	"runtime"

	"github.com/andrewpillar/thrall/errors"

	"github.com/pelletier/go-toml"
)

type Worker struct {
	Parallelism int

	Drivers []string

	Net struct {
		Listen string

		SSL struct {
			Listen string
			Cert   string
			Key    string
		}
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

	Objects struct {
		Type string
		Dir  string
	}

	Artifacts struct {
		Type string
		Dir  string
	}

	SSH struct {
		Key     string
		Timeout int
	}

	Qemu struct {
		Dir    string
		CPUs   int    `toml:"cpus"`
		Memory int
		Port   int
		User   string
	}

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
