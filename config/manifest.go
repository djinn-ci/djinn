package config

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"io"
	"strings"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/runner"

	"gopkg.in/yaml.v2"
)

// Manifest is the type that represents a manifest for a build. This details the
// driver to use, variables to set, objects to place, VCS repositories to clone
// and the actual commands to run and in what order.
type Manifest struct {
	Namespace     string             `yaml:",omitempty"`
	Driver        map[string]string  `yaml:",omitempty"`
	Env           []string           `yaml:",omitempty"`
	Objects       runner.Passthrough `yaml:",omitempty"`
	Sources       []Source           `yaml:",omitempty"`
	Stages        []string           `yaml:",omitempty"`
	AllowFailures []string           `yaml:"allow_failures,omitempty"`
	Jobs          []Job              `yaml:",omitempty"`
}

// Source is the type that represents a VCS repository in a manifest.
type Source struct {
	URL string
	Ref string
	Dir string
}

// Job is the type that represents a single job to be executed in a build.
type Job struct {
	Stage     string             `yaml:",omitempty"`
	Name      string             `yaml:",omitempty"`
	Commands  []string           `yaml:",omitempty"`
	Artifacts runner.Passthrough `yaml:",omitempty"`
}

// DecodeManifest takes the given io.Reader, and decodes its content into a
// Manifest, which is then returned.
func DecodeManifest(r io.Reader) (Manifest, error) {
	var m Manifest

	if err := yaml.NewDecoder(r).Decode(&m); err != nil {
		return m, errors.Err(err)
	}
	return m, nil
}

func UnmarshalManifest(b []byte) (Manifest, error) {
	var m Manifest

	err := yaml.Unmarshal(b, &m)
	return m, errors.Err(err)
}

func (m *Manifest) Scan(val interface{}) error {
	if val == nil {
		return nil
	}

	str, err := driver.String.ConvertValue(val)

	if err != nil {
		return errors.Err(err)
	}

	s, ok := str.(string)

	if !ok {
		return errors.New("expecred string value for manifest")
	}

	if len(s) == 0 {
		return nil
	}

	buf := bytes.NewBufferString(s)
	dec := yaml.NewDecoder(buf)

	return errors.Err(dec.Decode(m))
}

func (m *Manifest) String() string {
	b, err := yaml.Marshal(m)

	if err != nil {
		return ""
	}
	return strings.TrimSuffix(string(b), "\n")
}

func (m *Manifest) UnmarshalJSON(b []byte) error {
	var s string

	if err := json.Unmarshal(b, &s); err != nil {
		return form.UnmarshalError{
			Field: "manifest",
			Err:   err,
		}
	}
	return m.UnmarshalText([]byte(s))
}

func (m *Manifest) UnmarshalText(b []byte) error {
	tmp := struct {
		Namespace     string             `yaml:",omitempty"`
		Driver        map[string]string  `yaml:",omitempty"`
		Env           []string           `yaml:",omitempty"`
		Objects       runner.Passthrough `yaml:",omitempty"`
		Sources       []Source           `yaml:",omitempty"`
		Stages        []string           `yaml:",omitempty"`
		AllowFailures []string           `yaml:"allow_failures,omitempty"`
		Jobs          []Job              `yaml:",omitempty"`
	}{}

	if err := yaml.Unmarshal(b, &tmp); err != nil {
		return form.UnmarshalError{
			Field: "manifest",
			Err:   err,
		}
	}

	m.Namespace = tmp.Namespace
	m.Driver = tmp.Driver
	m.Env = tmp.Env
	m.Objects = tmp.Objects
	m.Sources = tmp.Sources
	m.Stages = tmp.Stages
	m.AllowFailures = tmp.AllowFailures
	m.Jobs = tmp.Jobs

	return nil
}

func (m *Manifest) Validate() error {
	switch m.Driver["type"] {
	case "docker":
		if m.Driver["image"] == "" {
			return errors.New("driver docker requies image")
		}
		if m.Driver["workspace"] == "" {
			return errors.New("driver docker requires workspace")
		}
	case "qemu":
		if m.Driver["image"] == "" {
			return errors.New("driver qemu requires image")
		}
	case "ssh":
		if m.Driver["address"] == "" {
			return errors.New("driver ssh requires address")
		}
	default:
		return errors.New("invalid driver specified")
	}
	return nil
}

func (m Manifest) Value() (driver.Value, error) {
	var buf bytes.Buffer
	yaml.NewEncoder(&buf).Encode(&m)
	return driver.Value(buf.String()), nil
}
