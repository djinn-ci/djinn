package core

import "github.com/andrewpillar/thrall/errors"

var (
	ErrAccessDenied      = errors.New("access denied")
	ErrBadHookData       = errors.New("unexpectec data from webhook")
	ErrBuildNotRunning   = errors.New("build is not running")
	ErrNamespaceTooDeep  = errors.New("namespace cannot exceed depth of 20")
	ErrNoManifest        = errors.New("no manifest could be found")
	ErrNotFound          = errors.New("not found")
	ErrUnsupportedDriver = errors.New("unsupported driver")
	ErrValidationFailed  = errors.New("validation failed")
)
