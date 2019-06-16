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

func (c *FileSystem) Collect(name string, r io.Reader) (int64, error) {
	dst := filepath.Join(c.dir, name)

	if err := os.MkdirAll(filepath.Dir(dst), os.FileMode(0755)); err != nil {
		return 0, errors.Err(err)
	}

	f, err := os.Create(dst)

	if err != nil {
		return 0, errors.Err(err)
	}

	defer f.Close()

	n, err := io.Copy(f, r)

	return n, errors.Err(err)
}
