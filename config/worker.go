package config

import (
	"io"
	"runtime"

	"github.com/andrewpillar/thrall/errors"

	"github.com/pelletier/go-toml"
)

// Worker represents the configuration used for the thrall-worker.
type Worker struct {
	Parallelism int
	Queue       string
	Timeout     string

	Crypto struct {
		Block string
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

	Images    Storage
	Artifacts Storage
	Objects   Storage

	Log struct {
		Level string
		File  string
	}
}

// DecodeWorker takes the given io.Reader, and decodes its content to a Worker,
// which is then returned.
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
