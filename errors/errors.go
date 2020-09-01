// Package errors implements some utility functions for giving detailed errors.
package errors

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

var skip = 1

type Slice []error

// Error implements the builtin error interface. This captures information
// about the underlying error itself, and where the error occurred.
type Error struct {
	// Err is the underlying error that occurred.
	Err error

	// Func is the name of the function caller that triggered the error.
	Func string

	// File is the name of the file where the error occured.
	File string

	// Line is the line number in the file where the error occurred.
	Line int
}

type errorStr string

// New returns a simple string error. This is equivalent to the errors.New
// function from the stdlib.
func New(s string) error {
	e := errorStr(s)
	return &e
}

// MultiError returns a concatenation of the given errors.
func MultiError(err ...error) error {
	e := Slice(err)
	return &e
}

// Cause recurses down the given error, if it is Error, to find the underlying
// Err that triggered it.
func Cause(err error) error {
	e, ok := err.(*Error)

	if ok {
		return Cause(e.Err)
	}
	return err
}

// Err wraps the given error in the context in which it occurred. If the given
// err is nil then nil is returned.
func Err(err error) error {
	if err == nil {
		return nil
	}

	pc, fname, l, _ := runtime.Caller(skip)
	pcFunc := runtime.FuncForPC(pc)

	funcName := ""

	if pcFunc != nil {
		funcName = pcFunc.Name()
	}

	parts := strings.SplitN(fname, "djinn", 2)

	return &Error{
		Err:  err,
		Func: funcName,
		File: strings.TrimPrefix(filepath.Join(parts[1:]...), string(os.PathSeparator)),
		Line: l,
	}
}

// Error returns the full "stacktrace" of the error using the context data
// about that error.
func (e *Error) Error() string {
	return fmt.Sprintf("%s - %s:%d: %s", path.Base(e.Func), e.File, e.Line, e.Err)
}

func (e *errorStr) Error() string { return string(*e) }

func (e Slice) Error() string {
	buf := &bytes.Buffer{}

	for _, err := range e {
		buf.WriteString(err.Error() + "\n")
	}
	return buf.String()
}
