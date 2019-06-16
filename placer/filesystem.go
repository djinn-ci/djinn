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

func (p *FileSystem) Place(name string, w io.Writer) (int64, error) {
	src := filepath.Join(p.dir, name)

	info, err := os.Stat(src)

	if err != nil {
		return 0, errors.Err(err)
	}

	if info.IsDir() {
		return 0, errors.Err(errors.New("cannot place directory as an object"))
	}

	f, err := os.Open(src)

	if err != nil {
		return 0, errors.Err(err)
	}

	defer f.Close()

	n, err := io.Copy(w, f)

	return n, errors.Err(err)
}
