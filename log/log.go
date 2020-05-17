// Package log provides a simple interface for logging.
package log

import (
	"io"
	"log"
	"os"
)

// Logger wraps the Printf, Println, Fatalf, and Fatal methods for logging.
type Logger interface {
	// Printf writes a formatted string to the Logger.
	Printf(format string, v ...interface{})

	// Println writes a line to the Logger.
	Println(v ...interface{})

	// Fatalf writes a formatted string to the Logger. It is expected for a
	// call to Fatalf to exit the program.
	Fatalf(format string, v ...interface{})

	// Fatal writes the given arguments to the Logger. It is expected for a
	// call to Fatalf to exit the program.
	Fatal(v ...interface{})
}

// Level defines the level a Logger can use.
type Level uint8

type logState struct {
	level  Level
	logger Logger
}

type logger struct {
	level Level
}

//go:generate stringer -type Level -linecomment
const (
	debug Level = iota // DEBUG
	info               // INFO
	err                // ERROR
)

var (
	// state is the global state of the Logger being used. This controls the
	// level being logged at, and the underlying Logger being used.
	state = logState{
		level:  info,
		logger: NewStdLog(os.Stdout),
	}

	levelsMap = map[string]Level{
		"debug": debug,
		"info":  info,
		"error": err,
	}

	Debug = &logger{debug}
	Info  = &logger{info}
	Error = &logger{err}
)

// NewStdLog returns a new log.Logger from the standard library using the given
// io.Writer for logging to.
func NewStdLog(w io.Writer) *log.Logger { return log.New(w, "", log.Ldate|log.Ltime|log.LUTC) }

// SetLevel sets the level for the Logger to use. If the given string is not a
// valid log level then the level is not changed.
func SetLevel(s string) {
	if l, ok := levelsMap[s]; ok {
		state.level = l
	}
}

// SetLogger sets the Logger implementation to use
func SetLogger(l Logger) { state.logger = l }

func (l *logger) Printf(format string, v ...interface{}) {
	if l.level < state.level {
		return
	}
	format = l.level.String() + " " + format
	state.logger.Printf(format, v...)
}

func (l *logger) Println(v ...interface{}) {
	if l.level < state.level {
		return
	}
	v = append([]interface{}{ l.level }, v...)
	state.logger.Println(v...)
}

func (l *logger) Fatalf(format string, v ...interface{}) {
	format = l.level.String() + " " + format
	state.logger.Fatalf(format, v...)
}

func (l *logger) Fatal(v ...interface{}) {
	v = append([]interface{}{ l.level, " " }, v...)
	state.logger.Fatal(v...)
}
