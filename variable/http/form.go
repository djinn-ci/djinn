package http

import (
	"regexp"

	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

type Form struct {
	Namespace namespace.Path
	Key       string
	Value     string
}

var _ webutil.Form = (*Form)(nil)

func (f Form) Fields() map[string]string {
	return map[string]string{
		"namespace": f.Namespace.String(),
		"key":       f.Key,
		"value":     f.Value,
	}
}

type Validator struct {
	UserID    int64
	Variables variable.Store
	Form      Form
}

var (
	_ webutil.Validator = (*Validator)(nil)

	rekey = regexp.MustCompile("^[\\D]+[a-zA-Z0-9_]+$")
)

func (v Validator) Validate(errs webutil.ValidationErrors) {
	if v.Form.Key == "" {
		errs.Add("key", webutil.ErrFieldRequired("Key"))
	}

	opts := []query.Option{
		query.Where("user_id", "=", query.Arg(v.UserID)),
		query.Where("key", "=", query.Arg(v.Form.Key)),
	}

	if v.Form.Namespace.Valid {
		_, n, err := v.Form.Namespace.ResolveOrCreate(v.Variables.Pool, v.UserID)

		if err != nil {
			if perr, ok := err.(*namespace.PathError); ok {
				errs.Add("namespace", perr)
				return
			}
			errs.Add("fatal", err)
			return
		}

		if err := n.IsCollaborator(v.Variables.Pool, v.UserID); err != nil {
			if errors.Is(err, namespace.ErrPermission) {
				errs.Add("namespace", err)
				return
			}
			errs.Add("fatal", err)
			return
		}
		opts[0] = query.Where("namespace_id", "=", query.Arg(n.ID))
	}

	_, ok, err := v.Variables.Get(opts...)

	if err != nil {
		errs.Add("fatal", err)
		return
	}

	if ok {
		errs.Add("key", webutil.ErrFieldExists("Key"))
	}

	if !rekey.Match([]byte(v.Form.Key)) {
		errs.Add("key", errors.New("Invalid variable key"))
	}

	if v.Form.Value == "" {
		errs.Add("value", webutil.ErrFieldRequired("Value"))
	}
}
