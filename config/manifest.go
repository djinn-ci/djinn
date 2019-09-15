package config

import (
	"io"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"gopkg.in/yaml.v2"
)

type Manifest struct {
	Driver map[string]string

	Env []string

	Objects runner.Passthrough
	Sources []Source

	Stages        []string
	AllowFailures []string `yaml:"allow_failures"`

	Jobs []Job
}

type Source struct {
	URL string
	Ref string
	Dir string
}

type Job struct {
	Stage     string
	Name      string
	Commands  []string
	Depends   []string
	Artifacts runner.Passthrough
}

func DecodeManifest(r io.Reader) (Manifest, error) {
	dec := yaml.NewDecoder(r)
	manifest := Manifest{}

	if err := dec.Decode(&manifest); err != nil {
		return manifest, errors.Err(err)
	}

	return manifest, nil
}

func (m Manifest) Validate() error {
	typ := m.Driver["type"]

	if typ == "" {
		return errors.New("driver type undefined")
	}

	switch typ {
		case "docker":
			for _, key := range []string{"image", "workspace"} {
				if m.Driver[key] == "" {
					return errors.New("driver type docker requires " + key)
				}
			}
		case "qemu":
			if m.Driver["image"] == "" {
				return errors.New("driver type qemu requires image")
			}
		case "ssh":
			for _, key := range []string{"address", "username"} {
				if m.Driver[key] == "" {
					return errors.New("driver type ssh requires " + key)
				}
			}
		default:
			return errors.New("unknown driver type " + typ)
	}

	return nil
}

// Source URLs can be in the format of:
//
//   [url] [ref] => [dir]
//
// This will correctly unmarshal the given string, and parse it accordingly. The ref, and dir
// parts of the string are optional. If not specified the ref will be master, and the dir will be
// the base of the url.
func (s *Source) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string

	if err := unmarshal(&str); err != nil {
		return err
	}

	s.Ref = "master"

	parts := strings.Split(str, "=>")

	if len(parts) > 1 {
		s.Dir = parts[1]
	}

	parts = strings.Split(strings.TrimPrefix(strings.TrimSuffix(parts[0], " "), " "), " ")

	if len(parts) > 1 {
		s.Ref = parts[1]
	}

	s.URL = parts[0]

	urlParts := strings.Split(s.URL, "/")

	s.Dir = urlParts[len(urlParts) - 1]

	return nil
}
