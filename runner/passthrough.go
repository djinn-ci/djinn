package runner

import (
	"path/filepath"
	"strings"

	"github.com/andrewpillar/thrall/errors"
)

// Passthrough represents files we want passing between the guest and host
// environments. This is a simple map, whereby the key is the source file and
// the value is the destination. Objects and artifacts are the two entities
// that can be passed from one environment to the next.
//
// Objects are passed from the host to the guest. The key for an object
// passthrough represents the source file on the host, and the value represents
// the destination on the guest environment.
//
// Artifacts are passed from the guest to the host. The key for an artifact
// passthrough represents the source file on the guest, and the value represents
// the destination on the host environment.
type Passthrough map[string]string

func NewPassthrough() Passthrough {
	return Passthrough(make(map[string]string))
}

// In the manifest YAML file passthrough is expected to be presented like so:
//
//   [source] => [destination]
//
// The [destination] is optional, and if not provided the based of the [source]
// will be used intstead.
func (p *Passthrough) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string

	if err := unmarshal(&str); err != nil {
		return errors.Err(err)
	}

	parts := strings.Split(str, "=>")

	key := strings.TrimSpace(parts[0])
	val := filepath.Base(key)

	if len(parts) > 1 {
		val = strings.TrimSpace(parts[1])
	}

	(*p)[key] = val

	return nil
}
