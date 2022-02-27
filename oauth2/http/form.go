package http

import (
	"net/url"

	"djinn-ci.com/oauth2"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

type AppForm struct {
	Name        string
	Description string
	HomepageURI string `schema:"homepage_uri"`
	RedirectURI string `schema:"redirect_uri"`
}

var _ webutil.Form = (*AppForm)(nil)

func (f AppForm) Fields() map[string]string {
	return map[string]string{
		"name":         f.Name,
		"description":  f.Description,
		"homepage_uri": f.HomepageURI,
		"redirect_uri": f.RedirectURI,
	}
}

type AppValidator struct {
	UserID int64
	Form   AppForm
	Apps   *oauth2.AppStore
	App    *oauth2.App
}

var _ webutil.Validator = (*AppValidator)(nil)

func (v AppValidator) Validate(errs webutil.ValidationErrors) {
	if v.Form.Name == "" {
		errs.Add("name", webutil.ErrFieldRequired("Name"))
	}

	if v.App == nil || (v.App != nil && v.Form.Name != v.App.Name) {
		_, ok, err := v.Apps.Get(
			query.Where("user_id", "=", query.Arg(v.UserID)),
			query.Where("name", "=", query.Arg(v.Form.Name)),
		)

		if err != nil {
			errs.Add("fatal", err)
			return
		}

		if ok {
			errs.Add("name", webutil.ErrFieldExists("Name"))
		}
	}

	if v.Form.HomepageURI == "" {
		errs.Add("homepage_uri", webutil.ErrFieldRequired("Homepage URI"))
	}

	if _, err := url.Parse(v.Form.HomepageURI); err != nil {
		errs.Add("homepage_uri", err)
	}

	if v.Form.RedirectURI == "" {
		errs.Add("redirect_uri", webutil.ErrFieldRequired("Redirect URI"))
	}

	if _, err := url.Parse(v.Form.RedirectURI); err != nil {
		errs.Add("redirect_uri", err)
	}
}

type AuthorizeForm struct {
	Authorize   bool
	ClientID    string `schema:"client_id"`
	Handle      string
	Password    string
	Scope       string
	State       string
	RedirectURI string `schema:"redirect_uri"`
}

var _ webutil.Form = (*AuthorizeForm)(nil)

func (f AuthorizeForm) Fields() map[string]string {
	return map[string]string{"handle": f.Handle}
}

type AuthorizeValidator struct {
	Form AuthorizeForm
}

var _ webutil.Validator = (*AuthorizeValidator)(nil)

func (v AuthorizeValidator) Validate(errs webutil.ValidationErrors) {
	if !v.Form.Authorize {
		return
	}

	if v.Form.Handle == "" {
		errs.Add("handle", webutil.ErrFieldRequired("Email or username"))
	}

	if v.Form.Password == "" {
		errs.Add("password", webutil.ErrFieldRequired("Password"))
	}
}

type TokenForm struct {
	Name  string
	Scope oauth2.Scope
}

var _ webutil.Form = (*TokenForm)(nil)

func (f TokenForm) Fields() map[string]string {
	return map[string]string{
		"name":  f.Name,
		"scope": f.Scope.String(),
	}
}

type TokenValidator struct {
	UserID int64
	Form   TokenForm
	Tokens oauth2.TokenStore
	Token  *oauth2.Token
}

var _ webutil.Validator = (*TokenValidator)(nil)

func (v TokenValidator) Validate(errs webutil.ValidationErrors) {
	if v.Form.Name == "" {
		errs.Add("name", webutil.ErrFieldRequired("Name"))
	}

	if v.Token == nil || (v.Token != nil && v.Form.Name != v.Token.Name) {
		_, ok, err := v.Tokens.Get(
			query.Where("user_id", "=", query.Arg(v.UserID)),
			query.Where("name", "=", query.Arg(v.Form.Name)),
		)

		if err != nil {
			errs.Add("fatal", err)
			return
		}

		if ok {
			errs.Add("name", webutil.ErrFieldExists("Name"))
		}
	}
}
