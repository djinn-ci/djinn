package http

import (
	"context"
	"net/url"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/oauth2"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"
)

type AppForm struct {
	Pool *database.Pool `schema:"-"`
	App  *oauth2.App    `schema:"-"`
	User *auth.User     `schema:"-"`

	Name        string
	Description string
	HomepageURI string `schema:"homepage_uri"`
	RedirectURI string `schema:"redirect_uri"`
}

var _ webutil.Form = (*AppForm)(nil)

func (f *AppForm) Fields() map[string]string {
	return map[string]string{
		"name":         f.Name,
		"description":  f.Description,
		"homepage_uri": f.HomepageURI,
		"redirect_uri": f.RedirectURI,
	}
}

func (f *AppForm) Validate(ctx context.Context) error {
	var v webutil.Validator

	v.Add("name", f.Name, webutil.FieldRequired)
	v.Add("name", f.Name, func(ctx context.Context, val any) error {
		name := val.(string)

		if f.App == nil || (f.App != nil && name != f.App.Name) {
			_, ok, err := oauth2.NewAppStore(f.Pool).Get(
				ctx,
				query.Where("user_id", "=", query.Arg(f.User.ID)),
				query.Where("name", "=", query.Arg(val)),
			)

			if err != nil {
				return err
			}

			if ok {
				return webutil.ErrFieldRequired
			}
		}
		return nil
	})

	v.Add("homepage_uri", f.HomepageURI, webutil.FieldRequired)
	v.Add("homepage_uri", f.HomepageURI, func(ctx context.Context, val any) error {
		s := val.(string)

		if _, err := url.Parse(s); err != nil {
			return err
		}
		return nil
	})

	v.Add("redirect_uri", f.RedirectURI, webutil.FieldRequired)
	v.Add("redirect_uri", f.RedirectURI, func(ctx context.Context, val any) error {
		s := val.(string)

		if _, err := url.Parse(s); err != nil {
			return err
		}
		return nil
	})

	errs := v.Validate(ctx)

	return errs.Err()
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

func (f *AuthorizeForm) Fields() map[string]string {
	return map[string]string{"handle": f.Handle}
}

func (f *AuthorizeForm) Validate(ctx context.Context) error {
	if !f.Authorize {
		return nil
	}

	var v webutil.Validator

	v.Add("handle", f.Handle, webutil.FieldRequired)
	v.Add("password", f.Password, webutil.FieldRequired)

	errs := v.Validate(ctx)

	return errs.Err()
}

type TokenForm struct {
	Pool  *database.Pool `schema:"-"`
	Token *oauth2.Token  `schema:"-"`
	User  *auth.User     `schema:"-"`

	Name  string
	Scope oauth2.Scope
}

var _ webutil.Form = (*TokenForm)(nil)

func (f *TokenForm) Fields() map[string]string {
	return map[string]string{
		"name":  f.Name,
		"scope": f.Scope.String(),
	}
}

func (f *TokenForm) Validate(ctx context.Context) error {
	var v webutil.Validator

	v.Add("name", f.Name, webutil.FieldRequired)
	v.Add("name", f.Name, func(ctx context.Context, val any) error {
		name := val.(string)

		if f.Token == nil || (f.Token != nil && name != f.Token.Name) {
			_, ok, err := oauth2.NewTokenStore(f.Pool).Get(
				ctx,
				query.Where("user_id", "=", query.Arg(f.User.ID)),
				query.Where("name", "=", query.Arg(name)),
			)

			if err != nil {
				return err
			}

			if !ok {
				return webutil.ErrFieldRequired
			}
		}
		return nil
	})

	errs := v.Validate(ctx)

	return errs.Err()
}
