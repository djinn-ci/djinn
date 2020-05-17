package filestore

import (
	"io"
	"os"
	"path/filepath"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
)

// FileSystem implements the FileStore interface by using the functions
// available in the os package of the stdlib.
type FileSystem struct {
	// Dir specifies the directory of the filesystem that is being accessed.
	Dir string

	// Limit specifies the maximum size of files that can be placed in the
	// filesystem, or collected from the filesystem.
	Limit int64
}

var _ FileStore = (*FileSystem)(nil)

// NewFileSystem returns a new FileSystem based on the given config.Storage. If
// the given config.Storage cannot be statted, or does not specify a valid
// directory.
func NewFileSystem(cfg config.Storage) (*FileSystem, error) {
	info, err := os.Stat(cfg.Path)

	if err != nil {
		return nil, errors.Err(err)
	}

	if !info.IsDir() {
		return nil, errors.New("not a directory "+cfg.Path)
	}

	return &FileSystem{
		Dir:   cfg.Path,
		Limit: cfg.Limit,
	}, nil
}

// Collect reads the contents of the given io.Reader stream, and stores it
// in a file with the given name. If the Limit for the filesystem is set to
// 0, then no limit is placed on the number of bytes read from the stream.
func (fs *FileSystem) Collect(name string, r io.Reader) (int64, error) {
	dst := filepath.Join(fs.Dir, name)

	if err := os.MkdirAll(filepath.Dir(dst), os.FileMode(0755)); err != nil {
		return 0, errors.Err(err)
	}

	f, err := os.Create(dst)

	if err != nil {
		return 0, errors.Err(err)
	}

	defer f.Close()

	if fs.Limit == 0 {
		n, err := io.Copy(f, r)

		return n, errors.Err(err)
	}

	n, err := io.CopyN(f, r, fs.Limit)

	return n, errors.Err(err)
}

// Place writes the contents of the file for the given name into the given
// io.Writer. If the Limit for the filesystem is set to 0, then no limit is
// placed on the number of bytes written to the stream.
func (fs *FileSystem) Place(name string, w io.Writer) (int64, error) {
	src := filepath.Join(fs.Dir, name)

	info, err := os.Stat(src)

	if err != nil {
		return 0, errors.Err(err)
	}

	if info.IsDir() {
		return 0, errors.New("cannot place directory as an object")
	}

	f, err := os.Open(src)

	if err != nil {
		return 0, errors.Err(err)
	}

	defer f.Close()

	if fs.Limit == 0 {
		n, err := io.Copy(w, f)

		return n, errors.Err(err)
	}

	n, err := io.CopyN(w, f, fs.Limit)

	return n, errors.Err(err)
}

// Open calls os.Open passing through the full filepath of the filesystem Dir
// and the given name.
func (fs *FileSystem) Open(name string) (*os.File, error) {
	f, err := os.Open(filepath.Join(fs.Dir, name))
	return f, errors.Err(err)
}

// Open calls os.OpenFile passing through the full filepath of the filesystem
// Dir and the given name.
func (fs *FileSystem) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	f, err := os.OpenFile(filepath.Join(fs.Dir, name), flag, perm)
	return f, errors.Err(err)
}

// Stat calls os.Stat passing through the full filepath of the filesystem Dir
// and the given name.
func (fs *FileSystem) Stat(name string) (os.FileInfo, error) {
	info, err := os.Stat(filepath.Join(fs.Dir, name))
	return info, errors.Err(err)
}

// Remove calls os.Remove passing through the full filepath of the filesystem
// Dir and the given name.
func (fs *FileSystem) Remove(name string) error { return errors.Err(os.Remove(filepath.Join(fs.Dir, name))) }
