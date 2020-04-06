package runner

import (
	"database/sql/driver"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
)

type Status uint8

//go:generate stringer -type Status -linecomment
const (
	Queued Status = iota // queued
	Running              // running
	Passed               // passed
	PassedWithFailures   // passed_with_failures
	Failed               // failed
	Killed               // killed
	TimedOut             // timed_out
)

var statusMap = map[string]Status{
	"queued":               Queued,
	"running":              Running,
	"passed":               Passed,
	"passed_with_failures": PassedWithFailures,
	"failed":               Failed,
	"killed":               Killed,
	"timed_out":            TimedOut,
}

func (s *Status) Scan(val interface{}) error {
	b, err := model.Scan(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b) == 0 {
		(*s) = Status(0)
		return nil
	}
	return errors.Err(s.UnmarshalText(b))
}

func (s *Status) UnmarshalText(b []byte) error {
	var ok bool

	str := string(b)
	(*s), ok = statusMap[str]

	if !ok {
		return errors.New("unknown status "+str)
	}
	return nil
}

func (s Status) Value() (driver.Value, error) {
	return driver.Value(s.String()), nil
}
