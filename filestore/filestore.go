package filestore

import (
	"net/url"
	"os"

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

func New(s string) (FileStore, error) {
	u, err := url.Parse(s)

	if err != nil {
		return nil, errors.Err(err)
	}

	switch u.Scheme {
		case "file":
			fallthrough
		default:
			fs, err := NewFileSystem(u)

			return fs, errors.Err(err)
	}
}
