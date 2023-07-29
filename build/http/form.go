package http

import (
	"context"
	"encoding/json"
	"strings"
	"unicode"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/manifest"
	"djinn-ci.com/namespace"

	"github.com/andrewpillar/webutil/v2"
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

type TagForm struct {
	Tags tags
}

func (*TagForm) Fields() map[string]string      { return nil }
func (*TagForm) Validate(context.Context) error { return nil }

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

type Form struct {
	DB   *database.Pool `json:"-" schema:"-"`
	User *auth.User     `json:"-" schema:"-"`

	Drivers  map[string]struct{} `json:"-" schema:"-"`
	Manifest manifest.Manifest
	Comment  string
	Tags     tags
}

var _ webutil.Form = (*Form)(nil)

func (f *Form) Fields() map[string]string {
	return map[string]string{
		"manifest": f.Manifest.String(),
		"comment":  f.Comment,
		"tags":     f.Tags.String(),
	}
}

func driverValid(drivers map[string]struct{}) webutil.ValidatorFunc {
	return func(ctx context.Context, val any) error {
		if m, ok := val.(manifest.Manifest); ok {
			typ := m.Driver["type"]

			if typ == "qemu" {
				typ += "-" + m.Driver["arch"]
			}

			if _, ok := drivers[typ]; !ok {
				if driver.IsValid(typ) {
					return driver.ErrDisabled(typ)
				}
				return driver.ErrUnknown(typ)
			}
		}
		return nil
	}
}

func (f *Form) Validate(ctx context.Context) error {
	var v webutil.Validator

	v.WrapError(
		webutil.MapError(auth.ErrPermission, errors.New("cannot submit to namespace")),
		webutil.WrapFieldError,
	)

	v.Add("manifest", f.Manifest, webutil.FieldRequired)
	v.Add("manifest", f.Manifest, driverValid(f.Drivers))
	v.Add("manifest", f.Manifest, func(_ context.Context, v any) error {
		m := v.(manifest.Manifest)
		return m.Validate()
	})
	v.Add("manifest", f.Manifest.Namespace, func(ctx context.Context, v any) error {
		p, err := namespace.ParsePath(v.(string))

		if err != nil {
			return err
		}

		if p.Valid {
			_, n, err := p.Resolve(ctx, f.DB, f.User)

			if err != nil {
				return err
			}

			if err := n.IsCollaborator(ctx, f.DB, f.User); err != nil {
				return err
			}
		}
		return nil
	})

	errs := v.Validate(ctx)

	return errs.Err()
}
