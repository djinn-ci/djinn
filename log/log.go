package log

import (
	"io"
	"log"
	"os"
)

type Logger interface {
	Printf(format string, v ...interface{})

	Println(v ...interface{})

	Fatalf(format string, v ...interface{})

	Fatal(v ...interface{})
}

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
	// Global logger state for writing to the output stream.
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

func NewStdLog(w io.Writer) *log.Logger {
	return log.New(w, "", log.Ldate|log.Ltime|log.LUTC)
}

func SetLevel(s string) {
	if l, ok := levelsMap[s]; ok {
		state.level = l
	}
}

func SetLogger(l Logger) {
	state.logger = l
}

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
