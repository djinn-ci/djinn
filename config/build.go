package config

import (
	"io"

	"github.com/andrewpillar/thrall/errors"

	"gopkg.in/yaml.v2"
)

type Build struct {
	Driver struct {
		Type      string
		Image     string
		Workspace string
		Arch      string
	}

	Sources []struct {
		URL string
		Ref string
		Dir string
	}

	Stages        []string
	AllowFailures []string `yaml:"allow_failures"`

	Jobs []Job
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
