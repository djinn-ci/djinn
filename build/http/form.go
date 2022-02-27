package http

import (
	"encoding/json"
	"strings"
	"unicode"

	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/manifest"

	"github.com/andrewpillar/webutil"
)

type tags []string

func (t *tags) String() string { return strings.Join((*t), ",") }

func (t *tags) UnmarshalJSON(data []byte) error {
	ss := make([]string, 0)

	if err := json.Unmarshal(data, &ss); err != nil {
		return errors.Err(err)
	}

	(*t) = ss
	return nil
}

func isLetter(r rune) bool {
	return unicode.IsLetter(r) || '0' <= r && r <= '9' || r == '-' || r == '_' || r == '.' || r == '/'
}

func isTagValid(s string) bool {
	for _, r := range s {
		if !isLetter(r) {
			return false
		}
	}
	return true
}

func (t *tags) UnmarshalText(b []byte) error {
	str := string(b)
	parts := strings.Split(str, ",")

	set := make(map[string]struct{})

	(*t) = tags(make([]string, 0, len(parts)))

	for _, tag := range parts {
		tag = strings.TrimSpace(tag)

		if tag == "" {
			continue
		}

		if !isTagValid(tag) {
			continue
		}
		set[tag] = struct{}{}
	}

	for tag := range set {
		(*t) = append((*t), tag)
	}
	return nil
}

type Form struct {
	Manifest manifest.Manifest `json:"manifest" schema:"manifest"`
	Comment  string            `json:"comment"  schema:"comment"`
	Tags     tags              `json:"tags"     schema:"tags"`
}

var _ webutil.Form = (*Form)(nil)

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

// TagForm represents the data sent by a client for tagging a Build.
type TagForm struct {
	Tags tags `schema:"tags"`
}

var _ webutil.Form = (*TagForm)(nil)

// Files implements the webutil.Form interface. This is a stub method.
func (f *TagForm) Fields() map[string]string { return nil }

// UnmarshalJSON attempts to unmarshal the given byte slice into a slice of
// strings.
func (f *TagForm) UnmarshalJSON(data []byte) error {
	tags := make([]string, 0)

	if err := json.Unmarshal(data, &tags); err != nil {
		return errors.Err(err)
	}

	set := make(map[string]struct{})

	for _, name := range tags {
		if !isTagValid(name) {
			continue
		}
		set[name] = struct{}{}
	}

	f.Tags = make([]string, 0, len(set))

	for name := range set {
		f.Tags = append(f.Tags, name)
	}
	return nil
}

type Validator struct {
	Form    Form
	Drivers map[string]struct{}
}

func (v *Validator) Validate(errs webutil.ValidationErrors) {
	if v.Form.Manifest.String() == "{}" {
		errs.Add("manifest", webutil.ErrFieldRequired("Manifest"))
	}

	if err := v.Form.Manifest.Validate(); err != nil {
		errs.Add("manifest", err)
	}

	typ := v.Form.Manifest.Driver["type"]

	if typ == "qemu" {
		arch := "x86_64"
		typ += "-" + arch
	}

	if _, ok := v.Drivers[typ]; !ok {
		if driver.IsValid(typ) {
			errs.Add("manifest", driver.ErrDisabled(typ))
			return
		}
		errs.Add("manifest", driver.ErrUnknown(typ))
	}
}
