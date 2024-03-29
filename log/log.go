// Package log providers a simple logging implementation.
package log

import (
	"io"
	"log"
	"strings"

	"github.com/mcmathja/curlyq"
)

type Level uint8

//go:generate stringer -type Level -linecomment
const (
	Debug Level = iota + 1 // DEBUG
	Info                   // INFO
	Warn                   // WARN
	Error                  // ERROR
)

var Levels = map[string]Level{
	"debug": Debug,
	"info":  Info,
	"warn":  Warn,
	"error": Error,
}

// Logger is the type for logging information at different levels of severity.
// The Logger has three states representing each level that can be logged at,
// Debug, Info, and Error.
type Logger struct {
	closer io.Closer

	Debug state
	Info  state
	Warn  state
	Error state
}

// Queue is a logger that can be given to CurlyQ for logging information about
// the jobs being processed.
type Queue struct {
	*Logger
}

type state struct {
	logger *log.Logger
	level  Level
	actual Level
}

var _ curlyq.Logger = (*Queue)(nil)

// New returns a new Logger that will write to the given io.Writer. This will
// use the stdlib's logger with the log.Ldate, log.Ltime, and log.LUTC flags
// set. The default level of the returned Logger is info.
func New(wc io.WriteCloser) *Logger {
	defaultLevel := Info
	logger := log.New(wc, "", log.Ldate|log.Ltime|log.LUTC)

	return &Logger{
		closer: wc,
		Debug: state{
			logger: logger,
			level:  defaultLevel,
			actual: Debug,
		},
		Info: state{
			logger: logger,
			level:  defaultLevel,
			actual: Info,
		},
		Warn: state{
			logger: logger,
			level:  defaultLevel,
			actual: Warn,
		},
		Error: state{
			logger: logger,
			level:  defaultLevel,
			actual: Error,
		},
	}
}

// SetLevel sets the level of the logger. The level should be either "debug",
// "info", or "error". If the given string is none of these values then the
// logger's level will be unchanged.
func (l *Logger) SetLevel(s string) {
	if lvl, ok := Levels[strings.ToLower(s)]; ok {
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
	l.Warn.logger = logger
	l.Error.logger = logger
}

func (l *Logger) Close() error { return l.closer.Close() }

func (q Queue) Debug(v ...interface{}) { q.Logger.Debug.Println(v...) }
func (q Queue) Info(v ...interface{})  { q.Logger.Info.Println(v...) }
func (q Queue) Warn(v ...interface{})  { q.Logger.Warn.Println(v...) }
func (q Queue) Error(v ...interface{}) { q.Logger.Error.Println(v...) }

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
