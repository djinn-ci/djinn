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
type Passthrough struct {
	Values map[string]string
}

func (p *Passthrough) Set(key, val string) {
	if p.Values == nil {
		p.Values = make(map[string]string)
	}

	p.Values[key] = val
}

func (p Passthrough) MarshalYAML() (interface{}, error) {
	if p.Values == nil {
		return []string{}, nil
	}

	ss := make([]string, 0, len(p.Values))

	for k, v := range p.Values {
		ss = append(ss, k + " => " + v)
	}

	return ss, nil
}

// In the manifest YAML file passthrough is expected to be presented like so:
//
//   [source] => [destination]
//
// The [destination] is optional, and if not provided the based of the [source]
// will be used intstead.
func (p *Passthrough) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if p.Values == nil {
		p.Values = make(map[string]string)
	}

	ss := make([]string, 0)

	if err := unmarshal(&ss); err != nil {
		return errors.Err(err)
	}

	for _, s := range ss {
		parts := strings.Split(s, "=>")

		key := strings.TrimSpace(parts[0])
		val := filepath.Base(key)

		if len(parts) > 1 {
			val = strings.TrimSpace(parts[1])
		}

		p.Values[key] = val
	}

	return nil
}
