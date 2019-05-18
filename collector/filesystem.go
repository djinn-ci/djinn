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
	dst := filepath.Join(c.dir, name)

	if err := os.MkdirAll(filepath.Dir(dst), os.FileMode(0755)); err != nil {
		return errors.Err(err)
	}

	f, err := os.Create(dst)

	if err != nil {
		return errors.Err(err)
	}

	defer f.Close()

	_, err = io.Copy(f, r)

	return errors.Err(err)
}

func (c *FileSystem) Stat(name string) (os.FileInfo, error) {
	info, err := os.Stat(name)

	return info, errors.Err(err)
}
