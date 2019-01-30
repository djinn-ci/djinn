package config

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/andrewpillar/thrall/errors"

	"gopkg.in/yaml.v2"
)

type Build struct {
	Driver struct {
		Type      string
		Image     string
		Workspace string
		Arch      string
		Address   string
		Username  string
		Password  string
	}

	Sources       []Source
	Stages        []string
	AllowFailures []string `yaml:"allow_failures"`

	Jobs []Job
}

type Source struct {
	URL string
	Ref string
	Dir string
}

type Artifact struct {
	Source      string
	Destination string
}

type Job struct {
	Stage     string
	Name      string
	Commands  []string
	Depends   []string
	Artifacts []Artifact
}

func DecodeBuild(r io.Reader) (Build, error) {
	dec := yaml.NewDecoder(r)
	build := Build{}

	if err := dec.Decode(&build); err != nil {
		return build, errors.Err(err)
	}

	return build, nil
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

// Artifacts can be in the format of:
//
//   [source] => [destination]
//
// This will correctly unmarshal the given string, and parse it accordingly. The destination part
// is optional. If not specified then the base of the source will be used as the destination.
func (a *Artifact) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string

	if err := unmarshal(&str); err != nil {
		return err
	}

	parts := strings.Split(str, "=>")

	a.Source = strings.TrimPrefix(strings.TrimSuffix(parts[0], " "), " ")
	a.Destination = filepath.Base(a.Source)

	if len(parts) > 1 {
		a.Destination = strings.TrimPrefix(strings.TrimSuffix(parts[1], " "), " ")
	}

	return nil
}

func (b Build) Validate() error {
	if b.Driver.Type == "" {
		return errors.New("driver type undefined")
	}

	switch b.Driver.Type {
		case "docker":
			if b.Driver.Image == "" {
				return errors.New("driver type docker requires image")
			}

			if b.Driver.Workspace == "" {
				return errors.New("driver typ docker requires workspace")
			}
		case "qemu":
			if b.Driver.Image == "" {
				return errors.New("driver type qemu requires image")
			}
		case "ssh":
			if b.Driver.Address == "" {
				return errors.New("driver type ssh requires address")
			}

			if b.Driver.Username == "" {
				return errors.New("driver type ssh requires username")
			}
		default:
			return errors.New("unknown driver type " + b.Driver.Type)
	}

	return nil
}
