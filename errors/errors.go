package errors

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

var skip = 1

type Error struct {
	Err  error
	Func string
	File string
	Line int
}

type errorStr string

func New(s string) error {
	e := errorStr(s)
	return &e
}

func Cause(err error) error {
	e, ok := err.(*Error)

	if ok {
		return Cause(e.Err)
	}
	return err
}

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

	parts := strings.Split(fname, "thrall")

	return &Error{
		Err:  err,
		Func: funcName,
		File: strings.TrimPrefix(filepath.Join(parts[1:]...), string(os.PathSeparator)),
		Line: l,
	}
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s - %s:%d: %s", path.Base(e.Func), e.File, e.Line, e.Err)
}

func (e *errorStr) Error() string { return string(*e) }
