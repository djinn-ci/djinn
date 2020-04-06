package filestore

import (
	"os"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"
)

// FileStore provides a generic interface for storing files. Each
// implementation of a FileStore will also implement a Collector and placer.
type FileStore interface {
	runner.Collector
	runner.Placer

	Open(name string) (*os.File, error)

	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)

	Remove(name string) error
}

func New(cfg config.Storage) (FileStore, error) {
	switch cfg.Kind {
		case "file":
			fs, err := NewFileSystem(cfg)
			return fs, errors.Err(err)
		default:
			return nil, errors.New("unkown filestore kind "+cfg.Kind)
	}
}
