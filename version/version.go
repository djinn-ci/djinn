// Package version contains version information about the built Go binaries.
// The variables exported by this package will be set during the build time
// via the -ldflags with -X.
package version

var (
	Ref string // Ref is the output of "git rev-parse HEAD"
	Tag string // Tag is the current tag of the git repository
)
