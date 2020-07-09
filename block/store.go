package block

import (
	"io"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"
)

type limitedWriter struct {
	w io.Writer
	l int64
	n int64
}

// Record represents an arbitrary record of data that can be held in a Store. A
// Record can be used as an io.Writer, io.Reader, io.Seeker, and io.Closer.
//
// io.Writer - A Record should operate as a normal implementation of io.Writer
// where it writes len(p) number of bytes into the underlying data source. If a
// Record has a limit on the number of bytes that can be written to it, then
// the Write method should return ErrWriteLimit when that limit is reached or
// exceeded. Any subsequent writes to a closed Record should return
// ErrRecordClosed.
//
// io.Reader - A Record should operate as a normal implementation of io.Reader
// where it reads len(p) number of bytes into p from the underlying data
// source. If a subsequent read is made to a closed Record, then Read should
// return ErrRecordClosed.
//
// io.Seeker - A Record should operate as a normal implementation of io.Seeker
// where it goes to the specified offset based off the specified whence. If a
// subsequent seek is made to a closed Record, then Seek should return
// ErrRecordClosed.
//
// io.Closer - When a Record is closed this should prevent subsequent reads,
// writes, and seeks from happening. This should also reset the value that is
// returned from a call to Len. Any subsequent calls to Close on a closed
// Record should return ErrRecordClosed.
type Record interface {
	io.Reader
	io.Writer
	io.Seeker
	io.Closer
}

// Store represents an arbitrary stoe of data. Each object within the Store is
// represented via the Record interface. Each store should also implement the
// runner.Collector and runner.Placer interfaces.
type Store interface {
	runner.Collector
	runner.Placer

	// Init initializes the Store for creating, retrieving, and removing
	// Records of data.
	Init() error

	// Create creates a new Record in the store with the given name. If a Record
	// of any given name already exists then ErrRecordExists should be returned.
	Create(string) (Record, error)

	// Open returns an existing Record from the Store with the given name. If
	// Record does not exist, then ErrRecordNotFound should be returned.
	Open(string) (Record, error)

	// Remove removes an existing Record from the Store with the given name. If
	// the Record does not exist, then ErrRecordNotFound should be returned.
	Remove(string) error
}

var (
	ErrRecordExists   = errors.New("record exists")
	ErrRecordNotFound = errors.New("record not found")
	ErrRecordClosed   = errors.New("record closed")
	ErrWriteLimit     = errors.New("write limit reached")
)

// NewLimitedWriter wraps the given io.Writer, and applies a limit of l to the
// number of bytes that can be written to it.
func NewLimitedWriter(w io.Writer, l int64) *limitedWriter {
	return &limitedWriter{
		w: w,
		l: l,
	}
}

// Write implements the io.Writer interface. If the number of bytes written to
// the underlying io.Writer reaches or exceeds the set limit, then
// ErrWriteLimit will be returned.
func (w *limitedWriter) Write(p []byte) (int, error) {
	var err error
	l := int64(len(p))

	if w.l == 0 {
		goto doWrite
	}

	if w.n >= w.l {
		return 0, ErrWriteLimit
	}

	if w.n + l == w.l {
		err = ErrWriteLimit
		goto doWrite
	}

	if l > w.l {
		p = p[:w.l]
		err = ErrWriteLimit
	}

doWrite:
	n, werr := w.w.Write(p)
	w.n += int64(n)

	if werr != nil {
		return n, werr
	}
	return n, err
}
