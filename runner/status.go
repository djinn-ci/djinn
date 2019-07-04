package runner

import (
	"database/sql/driver"

	"github.com/andrewpillar/thrall/errors"
)

type Status uint8

const (
	Queued Status = iota
	Running
	Passed
	PassedWithFailures
	Failed
	Killed
	TimedOut
)

func scan(val interface{}) ([]byte, error) {
	if val == nil {
		return []byte{}, nil
	}

	str, err := driver.String.ConvertValue(val)

	if err != nil {
		return []byte{}, errors.Err(err)
	}

	b, ok := str.([]byte)

	if !ok {
		return []byte{}, errors.Err(errors.New("failed to scan bytes"))
	}

	return b, nil
}

func (s *Status) Scan(val interface{}) error {
	b, err := scan(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b) == 0 {
		(*s) = Status(0)
		return nil
	}

	return errors.Err(s.UnmarshalText(b))
}

func (s Status) Signal() {}

func (s Status) String() string {
	switch s {
		case Queued:
			return "queued"
		case Running:
			return "running"
		case Passed:
			return "passed"
		case Failed:
			return "failed"
		case PassedWithFailures:
			return "passed with failures"
		case Killed:
			return "killed"
		case TimedOut:
			return "timed out"
		default:
			return "unknown status"
	}
}

func (s *Status) UnmarshalText(b []byte) error {
	str := string(b)

	switch str {
		case "queued":
			(*s) = Queued
			return nil
		case "running":
			(*s) = Running
			return nil
		case "failed":
			(*s) = Failed
			return nil
		case "passed":
			(*s) = Passed
			return nil
		case "passed_with_failures":
			(*s) = PassedWithFailures
			return nil
		case "killed":
			(*s) = Killed
			return nil
		case "timed_out":
			(*s) = TimedOut
			return nil
		default:
			return errors.Err(errors.New("unknown status " + str))
	}
}

func (s Status) Value() (driver.Value, error) {
	switch s {
		case Queued:
			return driver.Value("queued"), nil
		case Running:
			return driver.Value("running"), nil
		case Passed:
			return driver.Value("passed"), nil
		case Failed:
			return driver.Value("failed"), nil
		case PassedWithFailures:
			return driver.Value("passed_with_failures"), nil
		default:
			return driver.Value(""), errors.Err(errors.New("unknown status"))
	}
}
