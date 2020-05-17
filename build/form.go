package build

import (
	"strings"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/form"
)

type tags []string

type Form struct {
	Manifest config.Manifest `schema:"manifest"`
	Comment  string          `schema:"comment"`
	Tags     tags            `schema:"tags"`
}

type TagForm struct {
	Tags tags `schema:"tags"`
}

var (
	_ form.Form = (*Form)(nil)
	_ form.Form = (*TagForm)(nil)
)

// UnmarshalText parses the slice of bytes as a comma separated string. Each
// delineation will be treated as a separate tag, and appended to the
// underlying string slice.
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

// String returns the comma concatenated string of tags.
func (t *tags) String() string { return strings.Join((*t), ",") }

// Fields returns the fields of the build form. If the manifest is empty then
// return an empty string instead of a pair of {}.
func (f Form) Fields() map[string]string {
	manifest := f.Manifest.String()

	if manifest == "{}" {
		manifest = ""
	}

	return map[string]string{
		"manifest": manifest,
		"comment":  f.Comment,
		"tags":     f.Tags.String(),
	}
}

// Validate checks to see if there is a manifest, and if that manifest has the
// bare minimum for a build to be submitted.
func (f Form) Validate() error {
	errs := form.NewErrors()

	if f.Manifest.String() == "{}" {
		errs.Put("manifest", form.ErrFieldRequired("Build manifest"))
	}
	if err := f.Manifest.Validate(); err != nil {
		errs.Put("manifest", form.ErrFieldInvalid("Build manifest", err.Error()))
	}
	return errs.Err()
}

// Fields is a stub method to statisfy the form.Form interface. It returns an
// empty map.
func (f TagForm) Fields() map[string]string { return map[string]string{} }

// Validate is a stub method to satisfy the form.Form interface. It returns
// nil.
func (f TagForm) Validate() error { return nil }
