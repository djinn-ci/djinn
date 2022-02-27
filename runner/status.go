package runner

import (
	"database/sql/driver"

	"djinn-ci.com/errors"
)

type Status uint8

//go:generate stringer -type Status -linecomment
const (
	Queued             Status = iota // queued
	Running                          // running
	Passed                           // passed
	PassedWithFailures               // passed_with_failures
	Failed                           // failed
	Killed                           // killed
	TimedOut                         // timed_out
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

// Scan scans the given interface value into the current Status. If the value
// scans into an empty byte slice, then the Status is set to Queued, otherwise
// UnmarshalText is used to attempt to try and get the Status.
func (s *Status) Scan(val interface{}) error {
	v, err := driver.String.ConvertValue(val)

	if err != nil {
		return errors.Err(err)
	}

	str, ok := v.(string)

	if !ok {
		return errors.New("runner: could not type assert Status to string")
	}

	if str == "" {
		return nil
	}

	if err := s.UnmarshalText([]byte(str)); err != nil {
		return errors.Err(err)
	}
	return nil
}

// UnmarshalText unmarshals the given byte slice into the current Status, if it
// is a valid Status for the Runner. If the byte slice is of an unknown Status
// then the error "unknown status" is returned.
func (s *Status) UnmarshalText(b []byte) error {
	var ok bool

	str := string(b)
	(*s), ok = statusMap[str]

	if !ok {
		return errors.New("runner: unknown status " + str)
	}
	return nil
}

// Value returns the underlying string value for the current status to be used
// for database insertion.
func (s Status) Value() (driver.Value, error) { return driver.Value(s.String()), nil }
