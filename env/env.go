// Package env loads in the environment variables that Djinn expects and makes
// use of into package globals.
package env

import "os"

var (
	DJINN_API_DOCS  string
	DJINN_USER_DOCS string
)

// Load will load in the environment variables Djinn into their respective
// package level variables. This will only be done if the variables have not
// been previously set.
func Load() {
	envvars := map[string]*string{
		"DJINN_API_DOCS":  &DJINN_API_DOCS,
		"DJINN_USER_DOCS": &DJINN_USER_DOCS,
	}

	for k, p := range envvars {
		if (*p) == "" {
			(*p) = os.Getenv(k)
		}
	}
}
