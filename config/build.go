package config

import (
	"io"
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

type Job struct {
	Stage     string
	Name      string
	Commands  []string
	Depends   []string
	Artifacts []string
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
// parts of the string are optional.
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
