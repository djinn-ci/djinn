package fs

import (
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/andrewpillar/djinn/errors"
)

// Null provides an implementation of the Store interface for working with
// zero value Records of data. This is typically used for testing if you don't
// particularly care about what happens to the data being stored. This is the
// only implementation that won't return an error for the Place, Stat, Init,
// Create, Open, or Remove methods.
//
// Any reads that are performed on a Record returned from a Null Store will
// zero out the given byte slice.
type Null struct {
	w io.Writer
}

type nullRecord struct {
	w io.Writer
}

var (
	_ Store       = (*Null)(nil)
	_ Record      = (*nullRecord)(nil)
	_ os.FileInfo = (*nullRecord)(nil)
)

func NewNull() *Null {
	return &Null{
		w: ioutil.Discard,
	}
}

// Collect will copy everything from the given io.Reader to ioutil.Discard.
func (nl *Null) Collect(_ string, r io.Reader) (int64, error) {
	n, err := io.Copy(nl.w, r)
	return int64(n), errors.Err(err)
}

// Place does nothing.
func (nl *Null) Place(_ string, _ io.Writer) (int64, error) { return 0, nil }

// Stat will return the os.FileInfo of the /dev/null block device. This will
// always return the same information regardless of the host OS being run, the
// returned implementation of os.FileInfo is hardcoded.
func (nl *Null) Stat(_ string) (os.FileInfo, error) { return &nullRecord{w: ioutil.Discard}, nil }

// Init does nothing.
func (nl *Null) Init() error { return nil }

// Create returns a new null Record that will write everything to
// ioutil.Discard.
func (nl *Null) Create(_ string) (Record, error) { return &nullRecord{w: ioutil.Discard}, nil }

// Open returns a new null Record that will write everything to
// ioutil.Discard.
func (nl *Null) Open(_ string) (Record, error) { return &nullRecord{w: ioutil.Discard}, nil }

// Remove doesn nothing.
func (nl *Null) Remove(_ string) error { return nil }

func (r *nullRecord) Name() string       { return "/dev/null" }
func (r *nullRecord) Size() int64        { return 0 }
func (r *nullRecord) Mode() os.FileMode  { return os.FileMode(0666) }
func (r *nullRecord) ModTime() time.Time { return time.Now() }
func (r *nullRecord) IsDir() bool        { return false }
func (r *nullRecord) Sys() interface{}   { return nil }

func (r *nullRecord) Write(p []byte) (int, error) {
	n, err := r.w.Write(p)
	return n, errors.Err(err)
}

func (r *nullRecord) Seek(offset int64, _ int) (int64, error) { return offset, nil }

func (r *nullRecord) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

func (r *nullRecord) Close() error { return nil }
