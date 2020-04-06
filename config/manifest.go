package config

import (
	"bytes"
	"database/sql/driver"
	"io"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"gopkg.in/yaml.v2"
)

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

type Source struct {
	URL string
	Ref string
	Dir string
}

type Job struct {
	Stage     string             `yaml:",omitempty"`
	Name      string             `yaml:",omitempty"`
	Commands  []string           `yaml:",omitempty"`
	Depends   []string           `yaml:",omitempty"`
	Artifacts runner.Passthrough `yaml:",omitempty"`
}

func DecodeManifest(r io.Reader) (Manifest, error) {
	dec := yaml.NewDecoder(r)
	manifest := Manifest{}

	if err := dec.Decode(&manifest); err != nil {
		return manifest, errors.Err(err)
	}

	return manifest, nil
}

func UnmarshalManifest(b []byte) (Manifest, error) {
	manifest := Manifest{}

	err := yaml.Unmarshal(b, &manifest)

	return manifest, errors.Err(err)
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
		return errors.Err(errors.New("expected string value for manifest"))
	}

	if len(s) == 0 {
		return nil
	}

	buf := bytes.NewBufferString(s)
	dec := yaml.NewDecoder(buf)

	return errors.Err(dec.Decode(m))
}

func (m Manifest) String() string {
	b, err := yaml.Marshal(m)

	if err != nil {
		return ""
	}
	return strings.TrimSuffix(string(b), "\n")
}

func (m Manifest) Value() (driver.Value, error) {
	buf := &bytes.Buffer{}

	enc := yaml.NewEncoder(buf)
	enc.Encode(&m)

	return driver.Value(buf.String()), nil
}

func (m *Manifest) UnmarshalText(b []byte) error {
	tmp := struct{
		Namespace     string             `yaml:",omitempty"`
		Driver        map[string]string  `yaml:",omitempty"`
		Env           []string           `yaml:",omitempty"`
		Objects       runner.Passthrough `yaml:",omitempty"`
		Sources       []Source           `yaml:",omitempty"`
		Stages        []string           `yaml:",omitempty"`
		AllowFailures []string           `yaml:"allow_failures,omitempty"`
		Jobs          []Job              `yaml:",omitempty"`
	}{}

	err := yaml.Unmarshal(b, &tmp)

	m.Namespace = tmp.Namespace
	m.Driver = tmp.Driver
	m.Env = tmp.Env
	m.Objects = tmp.Objects
	m.Sources = tmp.Sources
	m.Stages = tmp.Stages
	m.AllowFailures = tmp.AllowFailures
	m.Jobs = tmp.Jobs

	return errors.Err(err)
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

func (s Source) MarshalYAML() (interface{}, error) {
	urlParts := strings.Split(s.URL, "/")

	ref := "master"
	dir := urlParts[len(urlParts) - 1]

	if s.Ref != "" {
		ref = s.Ref
	}

	if s.Dir != "" {
		dir = s.Dir
	}

	return s.URL + " " + ref + " => " + dir, nil
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
