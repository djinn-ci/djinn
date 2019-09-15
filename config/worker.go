package config

import (
	"io"
	"runtime"

	"github.com/andrewpillar/thrall/errors"

	"github.com/pelletier/go-toml"
)

type Worker struct {
	Parallelism int
	Driver      string
	Timeout     string

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

	Artifacts FileStore
	Objects   FileStore

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
