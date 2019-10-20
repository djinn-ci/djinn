package form

import (
	"strings"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
)

type tags []string

type Build struct {
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
	m["manifest"] = f.Manifest
	m["comment"] = f.Comment
	m["tags"] = f.Tags.String()

	return m
}

func (f Build) Validate() error {
	errs := NewErrors()

	if f.Manifest == "" {
		errs.Put("manifest", ErrFieldRequired("Build manifest"))
	}

	m, err := config.DecodeManifest(strings.NewReader(f.Manifest))

	if err != nil {
		errs.Put("manifest", ErrFieldInvalid("Build manifest", errors.Cause(err).Error()))
	}

	if err := m.Validate(); err != nil {
		errs.Put("manifest", ErrFieldInvalid("Build manifest", err.Error()))
	}

	return errs.Err()
}
