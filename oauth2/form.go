package oauth2

import (
	"strings"

	"github.com/andrewpillar/thrall/form"
)

type AppForm struct {
	Name        string `schema:"name"`
	Description string `schema:"description"`
	Domain      string `schema:"domain"`
	HomepageURI string `schema:"homepage_uri"`
	RedirectURI string `schema:"redirect_uri"`
}

type AuthorizeForm struct {
	Authenticate bool   `schema:"authenticate"`
	ClientID     string `schema:"client_id"`
	Handle       string `schema:"handle"`
	Password     string `schema:"password"`
	Scope        string `schema:"scope"`
	State        string `schema:"state"`
	RedirectURI  string `schema:"redirect_uri"`
}

type TokenForm struct {
	Tokens TokenStore `schema:"-"`
	Token  *Token     `schema:"-"`
	Name   string     `schema:"name"`
	Scope  []string   `schema:"scope[]"`
}

var (
	_ form.Form = (*AppForm)(nil)
	_ form.Form = (*AuthorizeForm)(nil)
	_ form.Form = (*TokenForm)(nil)
)

func (f AppForm) Fields() map[string]string {
	return map[string]string{
		"name":         f.Name,
		"description":  f.Description,
		"domain":       f.Domain,
		"homepage_uri": f.HomepageURI,
		"redirect_uri": f.RedirectURI,
	}
}

func (f AppForm) Validate() error {
	errs := form.NewErrors()
	return errs.Err()
}

func (f AuthorizeForm) Fields() map[string]string {
	return map[string]string{
		"handle": f.Handle,
	}
}

func (f AuthorizeForm) Validate() error {
	errs := form.NewErrors()
	return errs.Err()
}

func (f TokenForm) Fields() map[string]string {
	return map[string]string{
		"name":  f.Name,
		"scope": strings.Join(f.Scope, " "),
	}
}

func (f TokenForm) Validate() error {
	errs := form.NewErrors()
	return errs.Err()
}
