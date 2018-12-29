package collector

import (
	"io"
	"os"
	"path/filepath"

	"github.com/andrewpillar/thrall/errors"
)

type FileSystem struct {
	dir string
}

func NewFileSystem(dir string) *FileSystem {
	return &FileSystem{dir: dir}
}

func (c *FileSystem) Collect(name string, r io.Reader) error {
	f, err := os.Create(filepath.Join(c.dir, name))

	if err != nil {
		return errors.Err(err)
	}

	defer f.Close()

	_, err = io.Copy(f, r)

	return errors.Err(err)
}
