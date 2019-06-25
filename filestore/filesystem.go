package filestore

import (
	"io"
	"os"
	"path/filepath"

	"github.com/andrewpillar/thrall/errors"
)

func (fs *FileSystem) Collect(name string, r io.Reader) (int64, error) {
	dst := filepath.Join(fs.dir, name)

	if err := os.MkdirAll(filepath.Dir(dst), os.FileMode(0755)); err != nil {
		return 0, errors.Err(err)
	}

	f, err := os.Create(dst)

	if err != nil {
		return 0, errors.Err(err)
	}

	defer f.Close()

	if fs.limit == 0 {
		n, err := io.Copy(f, r)

		return n, errors.Err(err)
	}

	n, err := io.CopyN(f, r, fs.limit)

	return n, errors.Err(err)
}

func (fs *FileSystem) Place(name string, w io.Writer) (int64, error) {
	src := filepath.Join(fs.dir, name)

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

	if fs.limit == 0 {
		n, err := io.Copy(w, f)

		return n, errors.Err(err)
	}

	n, err := io.CopyN(w, f, fs.limit)

	return n, errors.Err(err)
}

func (fs *FileSystem) Open(name string) (*os.File, error) {
	f, err := os.Open(filepath.Join(fs.dir, name))

	return f, errors.Err(err)
}

func (fs *FileSystem) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	f, err := os.OpenFile(filepath.Join(fs.dir, name), flag, perm)

	return f, errors.Err(err)
}

func (fs *FileSystem) Stat(name string) (os.FileInfo, error) {
	info, err := os.Stat(filepath.Join(fs.dir, name))

	return info, errors.Err(err)
}

func (fs *FileSystem) Remove(name string) error {
	return errors.Err(os.Remove(filepath.Join(fs.dir, name)))
}
