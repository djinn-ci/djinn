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
	Comment   string `schema:"comment"`
	Tags      tags   `schema:"tags"`
}

func (t *tags) UnmarshalText(b []byte) error {
	str := string(b)

	parts := strings.Split(str, ",")

	(*t) = tags(make([]string, 0, len(parts)))

	for _, tag := range parts {
		tag = strings.TrimSpace(tag)

		if tag == "" {
			continue
		}

		(*t) = append((*t), tag)
	}

	return nil
}

func (t *tags) String() string {
	return strings.Join((*t), ",")
}

func (f Build) Fields() map[string]string {
	m := make(map[string]string)
	m["namespace"] = f.Namespace
	m["manifest"] = f.Manifest
	m["comment"] = f.Comment
	m["tags"] = f.Tags.String()

	return m
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
