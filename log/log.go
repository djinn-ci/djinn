package log

import (
	"io"
	"log"
	"os"
	"strings"
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

const (
	LevelDebug Level	= iota
	LevelInfo
	LevelError
)

var (
	Debug = &logger{LevelDebug}
	Info  = &logger{LevelInfo}
	Error = &logger{LevelError}

	// Global logger state for writing to the output stream.
	state = logState{
		level:  LevelInfo,
		logger: NewStdLog(os.Stdout),
	}
)

func toLevel(s string) Level {
	switch strings.ToLower(s) {
		case "debug":
			return LevelDebug
		case "info":
			return LevelInfo
		case "error":
			return LevelError
		default:
			return LevelInfo
	}
}

func NewStdLog(w io.Writer) *log.Logger {
	return log.New(w, "", log.Ldate|log.Ltime|log.LUTC)
}

func SetLevel(s string) {
	state.level = toLevel(s)
}

func SetLogger(l Logger) {
	state.logger = l
}

func (l Level) String() string {
	switch l {
		case LevelDebug:
			return "DEBUG"
		case LevelInfo:
			return "INFO "
		case LevelError:
			return "ERROR"
		default:
			return ""
	}
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
