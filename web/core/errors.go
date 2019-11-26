package core

import "github.com/andrewpillar/thrall/errors"

var (
	ErrAccessDenied         = errors.New("access denied")
	ErrBadHookData          = errors.New("unexpected data from webhook")
	ErrBuildNotRunning      = errors.New("build is not running")
	ErrInvalidManifest      = errors.New("manifest is not valid")
	ErrNamespaceTooDeep     = errors.New("namespace cannot exceed depth of 20")
	ErrNamespaceNameInvalid = errors.New("namespace could not be found")
	ErrNoManifest           = errors.New("no manifest could be found")
	ErrNotFound             = errors.New("not found")
	ErrUnsupportedDriver    = errors.New("unsupported driver")
	ErrValidationFailed     = errors.New("validation failed")
)
