package form

import (
	"strings"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
)

var (
	ErrBuildManifestRequired = errors.New("Build manifest can't be blank")
	ErrBuildManifestInvalid  = errors.New("Build manifest is not valid YAML")
)

type tags []string

type Build struct {
	Namespace string `schema:"namespace"`
	Manifest  string `schema:"manifest"`
	Tags      tags   `schema:"tags"`
}

func (t *tags) UnmarshalText(b []byte) error {
	str := string(b)

	parts := strings.Split(str, ",")

	(*t) = tags(make([]string, len(parts), len(parts)))

	for i, tag := range parts {
		(*t)[i] = tag
	}

	return nil
}

func (t *tags) String() string {
	return strings.Join((*t), ",")
}

func (f Build) Get(key string) string {
	if key == "namespace" {
		return f.Namespace
	}

	if key == "manifest" {
		return f.Manifest
	}

	if key == "tags" {
		return f.Tags.String()
	}

	return ""
}

func (f Build) Validate() error {
	errs := NewErrors()

	if f.Manifest == "" {
		errs.Put("manifest", ErrBuildManifestRequired)
	}

	m, err := config.DecodeManifest(strings.NewReader(f.Manifest))

	if err != nil {
		errs.Put("manifest", ErrBuildManifestInvalid)
	}

	if err := m.Validate(); err != nil {
		errs.Put("manifest", err)
	}

	return errs.Final()
}