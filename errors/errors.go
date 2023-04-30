package errors

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
)

var skip = 1

// Slice implements the builtin error interface. This captures a slice of
// errors.
type Slice []string

// Error implements the builtin error interface. This captures information
// about the underlying error itself, and where the error occurred.
type Error struct {
	Err  error  // Err is the underlying error that occured.
	Func string // Func is the name of the calling function that triggered the error.
	File string // File is the source file where the error occured.
	Line int    // Line is the line number in the source file where the error occurred.
}

type errorStr string

// New returns a simple string error. This is equivalent to the errors.New
// function from the stdlib.
func New(s string) error {
	e := errorStr(s)
	return &e
}

// Cause recurses down the given error, if it is Error, to find the underlying
// Err that triggered it.
func Cause(err error) error {
	if e, ok := err.(*Error); ok {
		return Cause(e.Err)
	}
	return err
}

func As(err error, target any) bool {
	return errors.As(err, target)
}

// Is reports whether any error in err's chain matches target. This calls
// errors.Is from the stdlib.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

type wrappedError struct {
	err error
	msg string
}

func (e wrappedError) Error() string { return e.msg }

func (e wrappedError) Unwrap() error { return e.err }

// Wrap annotates the given error with the given message.
func Wrap(err error, msg string) error {
	funcName, file, line := caller()

	return wrappedError{
		err: &Error{
			Err:  err,
			Func: funcName,
			File: file,
			Line: line,
		},
		msg: msg,
	}
}

func caller() (string, string, int) {
	pc, file, line, _ := runtime.Caller(skip + 1)
	pcFunc := runtime.FuncForPC(pc)

	funcName := ""

	if pcFunc != nil {
		funcName = pcFunc.Name()
	}
	return funcName, file, line
}

// Benign simply denotes an error that can be ignored. This would typically be
// used to exclude any error messages from logs, that you would want otherwise
// reported to a user, for example, validation errors.
type Benign string

func (e Benign) Error() string { return string(e) }

// Err wraps the given error in the context in which it occurred. If the given
// err is nil then nil is returned.
func Err(err error) error {
	if err == nil {
		return nil
	}

	funcName, file, line := caller()

	return &Error{
		Err:  err,
		Func: funcName,
		File: file,
		Line: line,
	}
}

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error. This calls errors.Unwrap
// from the stdlib.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

func Format(err error) string {
	if err == nil {
		return ""
	}

	e, ok := err.(*Error)

	if !ok {
		return err.Error()
	}
	return fmt.Sprintf("%s - %s:%d:\n%s", e.Func, e.File, e.Line, Format(e.Err))
}

// Error returns the full "stacktrace" of the error using the context data
// about that error.
func (e *Error) Error() string {
	return fmt.Sprintf("%s - %s:%d: %s", e.Func, e.File, e.Line, e.Err)
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.Err
}

func (e *errorStr) Error() string { return string(*e) }

// Err returns the underlying error value for the slice of errors, if the
// underlying slice contains errors.
func (e Slice) Err() error {
	if len(e) > 0 {
		return e
	}
	return nil
}

// Error returns a formatted string of the errors in the slice. Each error will
// be on a separate line in the returned string.
func (e Slice) Error() string {
	var buf bytes.Buffer

	json.NewEncoder(&buf).Encode(e)
	return buf.String()
}
