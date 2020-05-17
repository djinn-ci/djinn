package oauth2

import (
	"net/url"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"

	"github.com/andrewpillar/query"
)

type AppForm struct {
	Apps        *AppStore `schema:"-"`
	App         *App      `schema:"-"`
	Name        string    `schema:"name"`
	Description string    `schema:"description"`
	HomepageURI string    `schema:"homepage_uri"`
	RedirectURI string    `schema:"redirect_uri"`
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
	Tokens *TokenStore `schema:"-"`
	Token  *Token      `schema:"-"`
	Name   string      `schema:"name"`
	Scope  []string    `schema:"scope[]"`
}

var (
	_ form.Form = (*AppForm)(nil)
	_ form.Form = (*AuthorizeForm)(nil)
	_ form.Form = (*TokenForm)(nil)
)

// Fields returns a map of the name, description, homepage_uri, and
// redirect_uri fields for the current AppForm.
func (f AppForm) Fields() map[string]string {
	return map[string]string{
		"name":         f.Name,
		"description":  f.Description,
		"homepage_uri": f.HomepageURI,
		"redirect_uri": f.RedirectURI,
	}
}

// Validate checks to see if the Name field is present in the form, and is
// unique. This uniqueness check is skipped if an App is set on the form, and
// the Name field already matches that name (assuming it's being edited). The
// Homepage, and Redirect URIs are checked for presence too, and if they are
// valid URLs.
func (f AppForm) Validate() error {
	errs := form.NewErrors()

	if f.Name == "" {
		errs.Put("name", form.ErrFieldRequired("Name"))
	}

	checkUnique := true

	if f.App != nil {
		if f.App.Name == f.Name {
			checkUnique = false
		}
	}

	if checkUnique {
		a, err := f.Apps.Get(query.Where("name", "=", f.Name))

		if err != nil {
			return errors.Err(err)
		}

		if !a.IsZero() {
			errs.Put("name", form.ErrFieldExists("Name"))
		}
	}

	if f.HomepageURI == "" {
		errs.Put("homepage_uri", form.ErrFieldRequired("Homepage URI"))
	}

	if _, err := url.Parse(f.HomepageURI); err != nil {
		errs.Put("homepage_uri", form.ErrFieldInvalid("Homepage URI", err.Error()))
	}

	if f.RedirectURI == "" {
		errs.Put("redirect_uri", form.ErrFieldRequired("Redirect URI"))
	}

	if _, err := url.Parse(f.RedirectURI); err != nil {
		errs.Put("redirect_uri", form.ErrFieldInvalid("Redirect URI", err.Error()))
	}
	return errs.Err()
}

// Fields returns a map of fields for the current AuthorizeForm. This map will
// contain the Handle field of the current AuthorizeForm.
func (f AuthorizeForm) Fields() map[string]string {
	return map[string]string{
		"handle": f.Handle,
	}
}

// Validate checks to see if the Handle, and Password fields are present in the
// current AuthorizeForm.
func (f AuthorizeForm) Validate() error {
	errs := form.NewErrors()

	if f.Handle == "" {
		errs.Put("handle", form.ErrFieldRequired("Email or username"))
	}

	if f.Password == "" {
		errs.Put("password", form.ErrFieldRequired("Password"))
	}
	return errs.Err()
}

// Fields returns a map of the Name, and Scope fields in the current TokenForm.
// The Scope field is serialized into a space delimited string of the scopes.
func (f TokenForm) Fields() map[string]string {
	return map[string]string{
		"name":  f.Name,
		"scope": strings.Join(f.Scope, " "),
	}
}

// Validate cheks to see if the Name field is present in the form, and if it is
// unique. This uniquess check is skipped if a Token is set on the form, and
// Name field matches that.
func (f TokenForm) Validate() error {
	errs := form.NewErrors()

	if f.Name == "" {
		errs.Put("name", form.ErrFieldRequired("Name"))
	}

	checkUnique := true

	if f.Token != nil {
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
			errs.Put("name", form.ErrFieldExists("Name"))
		}
	}
	return errs.Err()
}
