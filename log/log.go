// Package log providers a simple logging implementation.
package log

import (
	"io"
	"log"
	"strings"
)

type level uint8

// Logger is the type for logging information at different levels of severity.
// The Logger has three states representing each level that can be logged at,
// Debug, Info, and Error.
type Logger struct {
	closer io.Closer

	Debug state
	Info  state
	Error state
}

type state struct {
	logger *log.Logger
	level  level
	actual level
}

//go:generate stringer -type level -linecomment
const (
	debug level = iota // DEBUG
	info               // INFO
	err                // ERROR
)

var levels = map[string]level{
	"debug": debug,
	"info":  info,
	"error": err,
}

// New returns a new Logger that will write to the given io.Writer. This will
// use the stdlib's logger with the log.Ldate, log.Ltime, and log.LUTC flags
// set. The default level of the returned Logger is info.
func New(wc io.WriteCloser) *Logger {
	defaultLevel := info
	logger := log.New(wc, "", log.Ldate|log.Ltime|log.LUTC)

	return &Logger{
		closer: wc,
		Debug: state{
			logger: logger,
			level:  defaultLevel,
			actual: debug,
		},
		Info: state{
			logger: logger,
			level:  defaultLevel,
			actual: info,
		},
		Error: state{
			logger: logger,
			level:  defaultLevel,
			actual: err,
		},
	}
}

// SetLevel sets the level of the logger. The level should be either "debug",
// "info", or "error". If the given string is none of these values then the
// logger's level will be unchanged.
func (l *Logger) SetLevel(s string) {
	if lvl, ok := levels[strings.ToLower(s)]; ok {
		l.Debug.level = lvl
		l.Info.level = lvl
		l.Error.level = lvl
	}
}

// SetWriter set's the io.Writer for the underlying logger.
func (l *Logger) SetWriter(w io.WriteCloser) {
	logger := log.New(w, "", log.Ldate|log.Ltime|log.LUTC)

	l.closer = w
	l.Debug.logger = logger
	l.Info.logger = logger
	l.Error.logger = logger
}

func (l *Logger) Close() error { return l.closer.Close() }

func (s *state) Printf(format string, v ...interface{}) {
	if s.actual < s.level {
		return
	}
	s.logger.Printf(s.actual.String()+" "+format, v...)
}

func (s *state) Println(v ...interface{}) {
	if s.actual < s.level {
		return
	}
	s.logger.Println(append([]interface{}{s.actual}, v...)...)
}

func (s *state) Fatalf(format string, v ...interface{}) {
	s.logger.Fatalf(s.actual.String()+" "+format, v...)
}

func (s *state) Fatal(v ...interface{}) {
	s.logger.Fatal(append([]interface{}{s.actual, " "}, v...)...)
}
