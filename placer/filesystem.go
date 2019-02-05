package placer

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

func (p *FileSystem) Place(name string, w io.Writer) error {
	info, err := os.Stat(name)

	if err != nil {
		return errors.Err(err)
	}

	if info.IsDir() {
		return errors.Err(errors.New("cannot place directory as an object"))
	}

	f, err := os.Open(filepath.Join(p.dir, name))

	if err != nil {
		return errors.Err(err)
	}

	defer f.Close()

	_, err = io.Copy(w, f)

	return errors.Err(err)
}
