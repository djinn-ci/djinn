// Package filestore provides the FileStore interface. Each implementation of
// this interface should also implement runner.Collector, and runner.Placer.
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

	// Open opens the given file for reading.
	Open(name string) (*os.File, error)

	// OpenFile opens the given file with the specified flag. If the file does
	// not exist, and the os.O_CREATE flag is passed, then a file should be
	// created with the given perm.
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)

	// Remove removes the given file, or directory.
	Remove(name string) error
}

// New returns a new FileStore interface based on the given config.Storage.
func New(cfg config.Storage) (FileStore, error) {
	switch cfg.Kind {
		case "file":
			fs, err := NewFileSystem(cfg)
			return fs, errors.Err(err)
		default:
			return nil, errors.New("unkown filestore kind "+cfg.Kind)
	}
}
