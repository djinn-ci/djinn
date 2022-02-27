package http

import (
	"regexp"

	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/object"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

type Form struct {
	File      *webutil.File
	Namespace namespace.Path
	Name      string
}

var _ webutil.Form = (*Form)(nil)

func (f Form) Fields() map[string]string {
	return map[string]string{
		"namespace": f.Namespace.String(),
		"name":      f.Name,
	}
}

type Validator struct {
	UserID  int64
	Objects *object.Store
	File    *webutil.FileValidator
	Form    *Form
}

var rename = regexp.MustCompile("^[a-zA-Z0-9\\._\\-]+$")

func (v Validator) Validate(errs webutil.ValidationErrors) {
	if v.Form.Name == "" {
		errs.Add("name", webutil.ErrFieldRequired("Name"))
	}

	if !rename.Match([]byte(v.Form.Name)) {
		errs.Add("name", errors.New("Name can only contain letters, numbers, and dashes"))
	}

	opts := []query.Option{
		query.Where("user_id", "=", query.Arg(v.UserID)),
		query.Where("name", "=", query.Arg(v.Form.Name)),
	}

	if v.Form.Namespace.Valid {
		_, n, err := v.Form.Namespace.ResolveOrCreate(v.Objects.Pool, v.UserID)

		if err != nil {
			errs.Add("fatal", err)
			return
		}
		opts[0] = query.Where("namespace_id", "=", query.Arg(n.ID))
	}

	_, ok, err := v.Objects.Get(opts...)

	if err != nil {
		errs.Add("fatal", err)
		return
	}

	if ok {
		errs.Add("name", webutil.ErrFieldExists("Name"))
	}
	v.File.Validate(errs)
}
