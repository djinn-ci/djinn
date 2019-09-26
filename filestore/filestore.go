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

func New(cfg config.FileStore) (FileStore, error) {
	switch cfg.Type {
		case "filesystem":
			return NewFileSystem(cfg.Path, cfg.Limit), nil
		default:
			return nil, errors.Err(errors.New("unknown filestore '" + cfg.Type + "'"))
	}
}

func NewFileSystem(dir string, limit int64) *FileSystem {
	return &FileSystem{
		dir:   dir,
		limit: limit,
	}
}
