package oauth2

import (
	"net/url"
	"strings"

	"github.com/andrewpillar/djinn/errors"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

// AppForm if the type that represents input data for creating a new App.
type AppForm struct {
	Apps        *AppStore `schema:"-"`
	App         *App      `schema:"-"`
	Name        string    `schema:"name"`
	Description string    `schema:"description"`
	HomepageURI string    `schema:"homepage_uri"`
	RedirectURI string    `schema:"redirect_uri"`
}

// AuthorizeForm is the type that represents input data of the OAuth
// authorization page, for when a user authorizes an App's access to their
// account.
type AuthorizeForm struct {
	Authorize   bool   `schema:"bool"`
	ClientID    string `schema:"client_id"`
	Handle      string `schema:"handle"`
	Password    string `schema:"password"`
	Scope       string `schema:"scope"`
	State       string `schema:"state"`
	RedirectURI string `schema:"redirect_uri"`
}

// TokenForm is the type that represents the input data for creating a new
// Token.
type TokenForm struct {
	Tokens *TokenStore `schema:"-"`
	Token  *Token      `schema:"-"`
	Name   string      `schema:"name"`
	Scope  []string    `schema:"scope[]"`
}

var (
	_ webutil.Form = (*AppForm)(nil)
	_ webutil.Form = (*AuthorizeForm)(nil)
	_ webutil.Form = (*TokenForm)(nil)
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
	errs := webutil.NewErrors()

	if f.Name == "" {
		errs.Put("name", webutil.ErrFieldRequired("Name"))
	}

	checkUnique := true

	if f.App != nil {
		if f.App.Name == f.Name {
			checkUnique = false
		}
	}

	if checkUnique {
		a, err := f.Apps.Get(query.Where("name", "=", query.Arg(f.Name)))

		if err != nil {
			return errors.Err(err)
		}

		if !a.IsZero() {
			errs.Put("name", webutil.ErrFieldExists("Name"))
		}
	}

	if f.HomepageURI == "" {
		errs.Put("homepage_uri", webutil.ErrFieldRequired("Homepage URI"))
	}

	if _, err := url.Parse(f.HomepageURI); err != nil {
		errs.Put("homepage_uri", webutil.ErrField("Homepage URI", err))
	}

	if f.RedirectURI == "" {
		errs.Put("redirect_uri", webutil.ErrFieldRequired("Redirect URI"))
	}

	if _, err := url.Parse(f.RedirectURI); err != nil {
		errs.Put("redirect_uri", webutil.ErrField("Redirect URI", err))
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
	if !f.Authorize {
		return nil
	}

	errs := webutil.NewErrors()

	if f.Handle == "" {
		errs.Put("handle", webutil.ErrFieldRequired("Email or username"))
	}

	if f.Password == "" {
		errs.Put("password", webutil.ErrFieldRequired("Password"))
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
	errs := webutil.NewErrors()

	if f.Name == "" {
		errs.Put("name", webutil.ErrFieldRequired("Name"))
	}

	checkUnique := true

	if f.Token != nil {
		if f.Token.Name == f.Name {
			checkUnique = false
		}
	}

	if checkUnique {
		t, err := f.Tokens.Get(query.Where("name", "=", query.Arg(f.Name)))

		if err != nil {
			return errors.Err(err)
		}

		if !t.IsZero() {
			errs.Put("name", webutil.ErrFieldExists("Name"))
		}
	}
	return errs.Err()
}
