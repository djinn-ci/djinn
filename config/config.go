// Package config provides structs and functions for working with TOML and YAML
// formatted configuration.
package config

import (
	"bytes"
	"database/sql/driver"
	"io"
	"runtime"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"github.com/pelletier/go-toml"

	"gopkg.in/yaml.v2"
)

// Server represents the configuration used for the thrall-server.
type Server struct {
	Host string

	Images    Storage
	Artifacts Storage
	Objects   Storage

	Net struct {
		Listen string

		SSL struct {
			Cert string
			Key  string
		}
	}

	Crypto struct {
		Hash  string
		Block string
		Salt  string
		Auth  string
	}

	Database Database

	Redis struct {
		Addr     string
		Password string
	}

	Log struct {
		Level string
		File  string
	}

	Drivers []struct {
		Type  string
		Queue string
	}

	Providers []struct {
		Name         string
		Secret       string
		Endpoint     string
		ClientID     string `toml:"client_id"`
		ClientSecret string `toml:"client_secret"`
	}
}

// Worker represents the configuration used for the thrall-worker.
type Worker struct {
	Parallelism int
	Queue       string
	Timeout     string

	Crypto struct {
		Block string
	}

	Redis struct {
		Addr     string
		Password string
	}

	Database struct {
		Addr     string
		Name     string
		Username string
		Password string
	}

	Images    Storage
	Artifacts Storage
	Objects   Storage

	Log struct {
		Level string
		File  string
	}
}

type Database struct {
	Addr     string
	Name     string
	Username string
	Password string
}

type Storage struct {
	Type  string
	Path  string
	Limit int64
}

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

var driverValidators = map[string]func(*toml.Tree) error{
	"ssh":    validateSSH,
	"docker": validateDocker,
	"qemu":   validateQEMU,
}

func validateSSH(tree *toml.Tree) error {
	for _, key := range []string{"timeout", "key"} {
		if !tree.Has(key) {
			return errors.New("ssh config missing property " + key)
		}
	}

	if _, ok := tree.Get("timeout").(int64); !ok {
		return errors.New("ssh timeout is not an integer")
	}

	if _, ok := tree.Get("key").(string); !ok {
		return errors.New("ssh key is not a string")
	}
	return nil
}

func validateDocker(_ *toml.Tree) error { return nil }

func validateQEMU(tree *toml.Tree) error {
	for _, key := range []string{"key", "disks", "cpus", "memory"} {
		if !tree.Has(key) {
			return errors.New("qemu config missing property " + key)
		}
	}

	if _, ok := tree.Get("key").(string); !ok {
		return errors.New("qemu key is not a string")
	}

	if _, ok := tree.Get("disks").(string); !ok {
		return errors.New("qemu disks is not an string")
	}

	if _, ok := tree.Get("cpus").(int64); !ok {
		return errors.New("qemu cpus is not an integer")
	}

	if _, ok := tree.Get("memory").(int64); !ok {
		return errors.New("qemu memory is not an integer")
	}
	return nil
}

// DecodeManifest takes the given io.Reader, and decodes its content into a
// Manifest, which is then returned.
func DecodeManifest(r io.Reader) (Manifest, error) {
	dec := yaml.NewDecoder(r)
	manifest := Manifest{}

	if err := dec.Decode(&manifest); err != nil {
		return manifest, errors.Err(err)
	}

	return manifest, nil
}

// UnmarshalManifest unmarshals the given bytes into Manifest, and returns it.
func UnmarshalManifest(b []byte) (Manifest, error) {
	manifest := Manifest{}

	err := yaml.Unmarshal(b, &manifest)
	return manifest, errors.Err(err)
}

// DecodeServer takes the given io.Reader, and decodes the its content into a
// Server, which is then returned.
func DecodeServer(r io.Reader) (Server, error) {
	dec := toml.NewDecoder(r)

	server := Server{}

	if err := dec.Decode(&server); err != nil {
		return server, errors.Err(err)
	}

	if server.Images.Type == "" {
		server.Images.Type = "file"
	}
	if server.Objects.Type == "" {
		server.Objects.Type = "file"
	}
	if server.Artifacts.Type == "" {
		server.Artifacts.Type = "file"
	}
	return server, nil
}

// DecodeWorker takes the given io.Reader, and decodes its content to a Worker,
// which is then returned.
func DecodeWorker(r io.Reader) (Worker, error) {
	dec := toml.NewDecoder(r)

	worker := Worker{}

	if err := dec.Decode(&worker); err != nil {
		return worker, errors.Err(err)
	}

	if worker.Parallelism == 0 {
		worker.Parallelism = runtime.NumCPU()
	}
	return worker, nil
}

// ValidateDriver takes the given toml.Tree and validates its configuration
// based off the driver the configuration is for.
func ValidateDrivers(tree *toml.Tree) error {
	keys := tree.Keys()

	if len(keys) == 0 {
		return errors.New("no drivers configured")
	}

	for _, key := range keys {
		if _, ok := driverValidators[key]; !ok {
			return errors.New("unknown driver configured: " + key)
		}

		subtree, ok := tree.Get(key).(*toml.Tree)

		if !ok {
			return errors.New("expected key-value configuration for driver: " + key)
		}
		if err := driverValidators[key](subtree); err != nil {
			return err
		}
	}
	return nil
}

// Scan unmarshals the given interface value, assuming the underlying value is
// that of a string.
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

	buf := bytes.NewBufferString(s)
	dec := yaml.NewDecoder(buf)
	return errors.Err(dec.Decode(m))
}

// String returns the marshalled string of the manifest.
func (m Manifest) String() string {
	b, err := yaml.Marshal(m)

	if err != nil {
		return ""
	}
	return strings.TrimSuffix(string(b), "\n")
}

// Value returns a marshalled version of the manifest to be inserted into the
// database.
func (m Manifest) Value() (driver.Value, error) {
	buf := &bytes.Buffer{}

	enc := yaml.NewEncoder(buf)
	enc.Encode(&m)
	return driver.Value(buf.String()), nil
}

// UnmarshalText unmarshals the given bytes into a temporary anonymous struct
// that matches Manifest. This is then copied into the underlying Manifest
// itself.
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

// Validate checks the manifest to see if the minimum configuration for a
// driver to execute is available.
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
		if m.Driver["address"] == "" {
			return errors.New("driver type ssh requires address")
		}
	default:
		return errors.New("unknown driver type " + typ)
	}
	return nil
}

// MarshalYAML marshals the YAML of a source URL into a string.
func (s Source) MarshalYAML() (interface{}, error) {
	urlParts := strings.Split(s.URL, "/")

	ref := "master"
	dir := urlParts[len(urlParts)-1]

	if s.Ref != "" {
		ref = s.Ref
	}

	if s.Dir != "" {
		dir = s.Dir
	}
	return s.URL + " " + ref + " => " + dir, nil
}

// UnmarshalYAML unmarshals the YAML for a source URL. Source URLs can be in
// the format of:
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

	s.Dir = urlParts[len(urlParts)-1]
	return nil
}
