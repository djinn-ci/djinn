package fs

import (
	"io"
	"os"
	"path/filepath"
	"strconv"

	"djinn-ci.com/errors"
)

// Filesystem provides an implementation of the Store interface for the
// operating system's native filesystem. This will allow the storing of the
// Records in a given directory of the filesystem with any given limit.
type Filesystem struct {
	dir string
	l   int64
}

type fileRecord struct {
	closed bool
	lw     *limitedWriter
	f      *os.File
	l      int64
}

var (
	_ Store  = (*Filesystem)(nil)
	_ Record = (*fileRecord)(nil)
)

// NewFilesystem returns a new Filesystem store using the given directory as
// the location to store Records. This will not put any limit on the size of
// Records that can be stored.
func NewFilesystem(dir string) *Filesystem {
	return NewFilesystemWithLimit(dir, 0)
}

// NewFilesystemWithLimit returns a new Filesystem store using the given
// directory as the location to store Records. This will put a limit on the size
// of the Records that can be stored. If the given limit is 0, then no limit is
// set.
func NewFilesystemWithLimit(dir string, l int64) *Filesystem {
	return &Filesystem{
		dir: dir,
		l:   l,
	}
}

func newFileRecord(f *os.File, l int64) *fileRecord {
	r := &fileRecord{
		f: f,
		l: l,
	}

	if l > 0 {
		r.lw = NewLimitedWriter(f, l)
	}
	return r
}

func (fs *Filesystem) checkDir() error {
	info, err := os.Stat(fs.dir)

	if err != nil {
		return err
	}

	if !info.IsDir() {
		return &PathError{
			Op:   "stat",
			Path: fs.dir,
			Err:  errors.New("not a directory"),
		}
	}
	return nil
}

func (fs *Filesystem) realpath(name string) string { return filepath.Join(fs.dir, name) }

func (fs *Filesystem) Limit() int64 { return fs.l }

// Collect reads the contents of the given io.Reader stream, and stores it
// in a file with the given name. If the Limit for the filesystem is set to
// 0, then no limit is placed on the number of bytes read from the stream.
func (fs *Filesystem) Collect(name string, r io.Reader) (int64, error) {
	f, err := os.Create(fs.realpath(name))

	if err != nil {
		return 0, errors.Err(err)
	}

	defer f.Close()

	lw := NewLimitedWriter(f, fs.l)

	n, err := io.Copy(lw, r)

	if err != nil {
		if errors.Is(err, ErrWriteLimit) {
			return n, nil
		}

		return n, &PathError{
			Op:   "collect",
			Path: fs.realpath(name),
			Err:  err,
		}
	}
	return n, nil
}

// Place writes the contents of the file for the given name into the given
// io.Writer. If the Limit for the filesystem is set to 0, then no limit is
// placed on the number of bytes written to the stream.
func (fs *Filesystem) Place(name string, w io.Writer) (int64, error) {
	src := fs.realpath(name)

	info, err := os.Stat(src)

	if err != nil {
		return 0, &PathError{
			Op:   "place",
			Path: src,
			Err:  err.(*os.PathError).Err,
		}
	}

	if info.IsDir() {
		return 0, &PathError{
			Op:   "place",
			Path: src,
			Err:  errors.New("cannot place directory as an object"),
		}
	}

	f, err := os.Open(src)

	if err != nil {
		return 0, &PathError{
			Op:   "place",
			Path: src,
			Err:  err.(*os.PathError).Err,
		}
	}

	defer f.Close()

	lw := NewLimitedWriter(w, fs.l)

	n, err := io.Copy(lw, f)

	if err != nil {
		if errors.Is(err, ErrWriteLimit) {
			return 0, nil
		}
		return 0, &PathError{
			Op:   "place",
			Path: src,
			Err:  err,
		}
	}
	return n, nil
}

// Init checks to see if the location in the filesystem is a directory and
// can be accessed.
func (fs *Filesystem) Init() error { return fs.checkDir() }

// Partition will create a new directory in the current Filesystem with the
// given number and returns it as a new Filesystem Store.
func (fs *Filesystem) Partition(number int64) (Store, error) {
	path := fs.realpath(strconv.FormatInt(number, 10))

	if err := os.MkdirAll(path, os.FileMode(0755)); err != nil {
		return nil, &PathError{
			Op:   "mkdir",
			Path: path,
			Err:  err,
		}
	}
	return NewFilesystemWithLimit(path, fs.l), nil
}

// Create will create a new file on the filesystem in the configured directory,
// and return a Record for that file.
func (fs *Filesystem) Create(name string) (Record, error) {
	if err := fs.checkDir(); err != nil {
		return nil, &PathError{
			Op:   "create",
			Path: fs.dir,
			Err:  err.(*os.PathError).Err,
		}
	}

	fname := fs.realpath(name)

	if err := os.MkdirAll(filepath.Dir(fname), os.FileMode(0750)); err != nil {
		return nil, &PathError{
			Op:   "create",
			Path: fs.realpath(name),
			Err:  ErrExist,
		}
	}

	if _, err := os.Stat(fname); err == nil {
		return nil, &PathError{
			Op:   "create",
			Path: fs.realpath(name),
			Err:  ErrExist,
		}
	}

	f, err := os.Create(fname)

	if err != nil {
		return nil, err
	}
	return newFileRecord(f, fs.l), nil
}

// Open will open an existing file on the filesystem in the configured
// directory, and return a Record for that file.
func (fs *Filesystem) Open(name string) (Record, error) {
	if err := fs.checkDir(); err != nil {
		return nil, err
	}

	f, err := os.Open(fs.realpath(name))

	if err != nil {
		return nil, err
	}
	return newFileRecord(f, fs.l), nil
}

func (fs *Filesystem) Stat(name string) (os.FileInfo, error) {
	info, err := os.Stat(fs.realpath(name))

	if err != nil {
		return nil, err
	}
	return info, nil
}

// Remove will remove an existing file on the filesystem in the configured
// directory.
func (fs *Filesystem) Remove(name string) error {
	if err := fs.checkDir(); err != nil {
		return err
	}

	fname := fs.realpath(name)

	if _, err := os.Stat(fname); err != nil {
		return err
	}

	if err := os.Remove(fname); err != nil {
		return err
	}
	return nil
}

func (r *fileRecord) Write(p []byte) (int, error) {
	if r.closed {
		return 0, &PathError{
			Op:   "write",
			Path: r.f.Name(),
			Err:  ErrClosed,
		}
	}

	if r.lw != nil {
		n, err := r.lw.Write(p)

		if err != nil {
			return n, &PathError{
				Op:   "write",
				Path: r.f.Name(),
				Err:  ErrWriteLimit,
			}
		}
		return n, nil
	}

	n, err := r.f.Write(p)

	if err != nil {
		return 0, &PathError{
			Op:   "write",
			Path: r.f.Name(),
			Err:  err,
		}
	}
	return n, nil
}

func (r *fileRecord) Read(p []byte) (int, error) {
	if r.closed {
		return 0, &PathError{
			Op:   "read",
			Path: r.f.Name(),
			Err:  ErrClosed,
		}
	}

	n, err := r.f.Read(p)

	if err != nil {
		return 0, &PathError{
			Op:   "read",
			Path: r.f.Name(),
			Err:  err,
		}
	}
	return n, nil
}

func (r *fileRecord) Seek(offset int64, whence int) (int64, error) {
	if r.closed {
		return 0, &PathError{
			Op:   "seek",
			Path: r.f.Name(),
			Err:  ErrClosed,
		}
	}

	off, err := r.f.Seek(offset, whence)

	if err != nil {
		return 0, &PathError{
			Op:   "seek",
			Path: r.f.Name(),
			Err:  err,
		}
	}
	return off, nil
}

func (r *fileRecord) Close() error {
	if r.closed {
		return &PathError{
			Op:   "close",
			Path: r.f.Name(),
			Err:  ErrClosed,
		}
	}
	r.closed = true
	return nil
}
