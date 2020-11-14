package build

import (
	"encoding/json"
	"strings"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/manifest"

	"github.com/andrewpillar/webutil"
)

type tags []string

// Form is the type that represents the input data for creating/submitting a
// build.
type Form struct {
	Manifest manifest.Manifest `schema:"manifest" json:"manifest"`
	Comment  string            `schema:"comment"  json:"comment"`
	Tags     tags              `schema:"tags"     json:"tags"`
}

// TagForm is the type that represents the input data for creating tags on a
// build.
type TagForm struct {
	Tags tags `schema:"tags"`
}

var (
	_ webutil.Form = (*Form)(nil)
	_ webutil.Form = (*TagForm)(nil)
)

// UnmarshalJSON parses the byte slice into a slice of strings.
func (t *tags) UnmarshalJSON(data []byte) error {
	ss := make([]string, 0)

	if err := json.Unmarshal(data, &ss); err != nil {
		return errors.Err(err)
	}

	(*t) = ss
	return nil
}

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
	errs := webutil.NewErrors()

	if f.Manifest.String() == "{}" {
		errs.Put("manifest", webutil.ErrFieldRequired("Build manifest"))
	}
	if err := f.Manifest.Validate(); err != nil {
		errs.Put("manifest", webutil.ErrField("Build manifest", err))
	}
	return errs.Err()
}

// Fields is a stub method to statisfy the form.Form interface. It returns an
// empty map.
func (f *TagForm) Fields() map[string]string { return map[string]string{} }

// Validate is a stub method to satisfy the form.Form interface. It returns
// nil.
func (f *TagForm) Validate() error { return nil }

// UnmarshalJSON will attempt to unmarshal the given byte slice into a slice of
// strings.
func (f *TagForm) UnmarshalJSON(data []byte) error {
	tags := make([]string, 0)

	if err := json.Unmarshal(data, &tags); err != nil {
		return errors.Err(err)
	}

	f.Tags = tags
	return nil
}
