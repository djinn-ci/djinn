package form

import (
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/model/types"

	"github.com/andrewpillar/query"
)

type Token struct {
	Tokens model.TokenStore
	Token  *model.Token

	Name  string   `schema:"name"`
	Scope []string `schema:"scope[]"`
}

func (f Token) Fields() map[string]string {
	return map[string]string{
		"name":  f.Name,
		"scope": strings.Join(f.Scope, " "),
	}
}

func (f Token) Validate() error {
	errs := NewErrors()

	if f.Name == "" {
		errs.Put("name", ErrFieldRequired("Name"))
	}

	checkUnique := true

	if f.Token != nil && !f.Token.IsZero() {
		if f.Token.Name == f.Name {
			checkUnique = false
		}
	}

	if checkUnique {
		t, err := f.Tokens.Get(query.Where("name", "=", f.Name))

		if err != nil {
			return errors.Err(err)
		}

		if !t.IsZero() {
			errs.Put("name", ErrFieldExists("name"))
		}
	}

	if len(f.Scope) > 0 {
		if _, err := types.UnmarshalScope(strings.Join(f.Scope, " ")); err != nil {
			return errors.Err(err)
		}
	}

	return errs.Err()
}
