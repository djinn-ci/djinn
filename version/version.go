// Package version contains version information about the built Go binaries.
// The variables exported by this package will be set during the build time
// via the -ldflags with -X.
package version

// Build is the build information fo the built Go binaries. The value of this
// will depend on whether or not this will built from a release branch or not.
var Build string
