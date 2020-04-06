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

func (t tags) String() string { return strings.Join(t, ",") }

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

func (f TagForm) Fields() map[string]string { return map[string]string{} }
func (f TagForm) Validate() error { return nil }
