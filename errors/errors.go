package errors

import (
	"fmt"
	"path"
	"runtime"
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

func Err(err error) error {
	if err == nil {
		return nil
	}

	pc, f, l, _ := runtime.Caller(skip)
	pcFunc := runtime.FuncForPC(pc)

	funcName := ""

	if pcFunc != nil {
		funcName = pcFunc.Name()
	}

	return &Error{
		Err:  err,
		Func: funcName,
		File: path.Base(f),
		Line: l,
	}
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s[%s:%d]:\n\t%s", path.Base(e.Func), path.Base(e.File), e.Line, e.Err)
}

func (e *errorStr) Error() string {
	return string(*e)
}

func Cause(err error) error {
	e, ok := err.(*Error)

	if ok {
		return Cause(e.Err)
	}

	return err
}
