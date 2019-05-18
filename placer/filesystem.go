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
	src := filepath.Join(p.dir, name)

	info, err := os.Stat(src)

	if err != nil {
		return errors.Err(err)
	}

	if info.IsDir() {
		return errors.Err(errors.New("cannot place directory as an object"))
	}

	f, err := os.Open(src)

	if err != nil {
		return errors.Err(err)
	}

	defer f.Close()

	_, err = io.Copy(w, f)

	return errors.Err(err)
}

func (p *FileSystem) Stat(name string) (os.FileInfo, error) {
	info, err := os.Stat(name)

	return info, errors.Err(err)
}
