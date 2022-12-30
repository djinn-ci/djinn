package manifest

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"io"
	"strings"

	"djinn-ci.com/errors"
	"djinn-ci.com/runner"

	"github.com/andrewpillar/webutil"

	"gopkg.in/yaml.v2"
)

// Driver is used for the driver block in the manifest YAML.
type Driver map[string]string

// Manifest is the type that represents a manifest for a build. This details the
// driver to use, variables to set, objects to place, VCS repositories to clone
// and the actual commands to run and in what order.
type Manifest struct {
	Namespace     string             `yaml:",omitempty"`
	Driver        Driver             `yaml:",omitempty"`
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

var (
	_ sql.Scanner   = (*Driver)(nil)
	_ driver.Valuer = (*Driver)(nil)
)

func base(s string) string {
	parts := strings.Split(s, "/")
	return parts[len(parts)-1]
}

func Decode(r io.Reader) (Manifest, error) {
	var m Manifest

	if err := yaml.NewDecoder(r).Decode(&m); err != nil {
		return m, errors.Err(err)
	}
	return m, nil
}

func Unmarshal(b []byte) (Manifest, error) {
	var m Manifest

	if err := yaml.Unmarshal(b, &m); err != nil {
		return m, errors.Err(err)
	}
	return m, nil
}

func (d Driver) Value() (driver.Value, error) { return driver.Value(d.String()), nil }

func (d *Driver) Scan(val interface{}) error {
	v, err := driver.String.ConvertValue(val)

	if err != nil {
		return errors.Err(err)
	}

	b, ok := v.([]byte)

	if !ok {
		return errors.New("manifest: could not type assert Driver to byte slice")
	}

	if b == nil {
		return nil
	}

	if err := json.Unmarshal(b, d); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (d *Driver) String() string {
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(d)
	return buf.String()
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
		return errors.New("expected string value for manifest")
	}

	if len(s) == 0 {
		return nil
	}

	dec := yaml.NewDecoder(strings.NewReader(s))

	return errors.Err(dec.Decode(m))
}

func (m *Manifest) String() string {
	b, err := yaml.Marshal(m)

	if err != nil {
		return ""
	}
	return strings.TrimSuffix(string(b), "\n")
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
		return webutil.UnmarshalError{
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

func (m *Manifest) UnmarshalJSON(b []byte) error {
	var s string

	if err := json.Unmarshal(b, &s); err != nil {
		return webutil.UnmarshalError{
			Field: "manifest",
			Err:   err,
		}
	}
	return m.UnmarshalText([]byte(s))
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
	case "os":
		break
	default:
		return errors.New("invalid driver specified " + m.Driver["type"])
	}
	return nil
}

func (m Manifest) Value() (driver.Value, error) {
	var buf bytes.Buffer
	yaml.NewEncoder(&buf).Encode(&m)
	return driver.Value(buf.String()), nil
}

func (s Source) MarshalYAML() (interface{}, error) {
	source := s.URL

	if s.Ref != "" {
		source += " " + s.Ref
	}

	if s.Dir != "" {
		source += " => " + s.Dir
	}
	return source, nil
}

// UnmarshalYAML unmarshals the YAML for a source URL. Source URLs can be in
// the format of:
//
//	[url] [ref] => [dir]
//
// This will correctly unmarshal the given string, and parse it accordingly. The ref, and dir
// parts of the string are optional. If not specified the ref will be master, and the dir will be
// the base of the url.
func (s *Source) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string

	if err := unmarshal(&str); err != nil {
		return err
	}

	i := strings.Index(str, " ")

	if i < 0 {
		i = len(str)
	}

	s.URL = str[:i]
	s.Dir = strings.TrimSuffix(base(s.URL), ".git")

	tmp := make([]rune, 0, len(str[i:]))

	for _, r := range str[i:] {
		if r == ' ' {
			continue
		}
		tmp = append(tmp, r)
	}

	str = string(tmp)

	i = strings.Index(str, "=>")

	if i >= 0 {
		s.Dir = str[i+2:]
		str = str[:i]
	}
	if len(str) > 0 {
		s.Ref = str
	}
	return nil
}
