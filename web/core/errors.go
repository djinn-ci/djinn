package core

import "github.com/andrewpillar/thrall/errors"

var (
	ErrAccessDenied      = errors.New("access denied")
	ErrBuildNotRunning   = errors.New("build is not running")
	ErrNotFound          = errors.New("not found")
	ErrUnsupportedDriver = errors.New("unsupported driver")
	ErrValidationFailed  = errors.New("validation failed")
)
