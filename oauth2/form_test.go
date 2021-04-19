package oauth2

import (
	"regexp"
	"testing"

	"djinn-ci.com/errors"

	"github.com/andrewpillar/webutil"

	"github.com/DATA-DOG/go-sqlmock"
)

func Test_AppForm(t *testing.T) {
	store, mock, close_ := appStore(t)
	defer close_()

	tests := []struct {
		form        AppForm
		errs        []string
		shouldError bool
	}{
		{
			AppForm{
				Apps:        store,
				Name:        "hl2",
				HomepageURI: "https://example.com",
				RedirectURI: "https://example.com/oauth/djinn",
			},
			[]string{},
			false,
		},
		{
			AppForm{
				Apps:        store,
				Name:        "hl2",
				HomepageURI: "https://example.com",
				RedirectURI: "https://example.com/oauth/djinn",
			},
			[]string{},
			false,
		},
		{
			AppForm{
				Apps:        store,
				HomepageURI: "https://example.com",
				RedirectURI: "https://example.com/oauth/djinn",
			},
			[]string{"name"},
			true,
		},
		{
			AppForm{
				Apps:        store,
				HomepageURI: "https://example.com",
				RedirectURI: "https://example.com/oauth/djinn",
			},
			[]string{"name"},
			true,
		},
		{
			AppForm{
				Apps:        store,
				RedirectURI: "https://example.com/oauth/djinn",
			},
			[]string{"name", "homepage_uri"},
			true,
		},
		{
			AppForm{Apps: store},
			[]string{"name", "homepage_uri", "redirect_uri"},
			true,
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(
			regexp.QuoteMeta("SELECT * FROM oauth_apps WHERE (name = $1)"),
		).WithArgs(test.form.Name).WillReturnRows(sqlmock.NewRows(appCols))

		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				cause := errors.Cause(err)

				ferrs, ok := cause.(*webutil.Errors)

				if !ok {
					t.Fatalf("test[%d] - expected error to be *webutil.Errors, it was %s\n", i, cause)
				}

				for _, err := range test.errs {
					if _, ok := (*ferrs)[err]; !ok {
						t.Errorf("test[%d] - expected field '%s' to be in *webutil.Errors\n", i, err)
					}
				}
				continue
			}
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}
	}
}

func Test_AuthorizeForm(t *testing.T) {
	tests := []struct {
		form        AuthorizeForm
		errs        []string
		shouldError bool
	}{
		{
			AuthorizeForm{Handle: "eli.vance@black-mesa.com", Password: "secret"},
			[]string{},
			false,
		},
		{
			AuthorizeForm{Handle: "eli.vance@black-mesa.com"},
			[]string{"password"},
			true,
		},
		{
			AuthorizeForm{},
			[]string{"handle", "password"},
			true,
		},
	}

	for i, test := range tests {
		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				cause := errors.Cause(err)

				ferrs, ok := cause.(*webutil.Errors)

				if !ok {
					t.Fatalf("test[%d] - expected error to be *webutil.Errors, it was %s\n", i, cause)
				}

				for _, err := range test.errs {
					if _, ok := (*ferrs)[err]; !ok {
						t.Errorf("test[%d] - expected field '%s' to be in *webutil.Errors\n", i, err)
					}
				}
				continue
			}
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}
	}
}

func Test_TokenForm(t *testing.T) {
	store, mock, close_ := tokenStore(t)
	defer close_()

	tests := []struct {
		form        TokenForm
		errs        []string
		shouldError bool
	}{
		{
			TokenForm{Tokens: store, Name: "api_token"},
			[]string{},
			false,
		},
		{
			TokenForm{Tokens: store},
			[]string{"name"},
			true,
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(
			regexp.QuoteMeta("SELECT * FROM oauth_tokens WHERE (name = $1)"),
		).WithArgs(test.form.Name).WillReturnRows(sqlmock.NewRows(tokenCols))

		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				cause := errors.Cause(err)

				ferrs, ok := cause.(*webutil.Errors)

				if !ok {
					t.Fatalf("test[%d] - expected error to be *webutil.Errors, is was '%s'\n", i, cause)
				}

				for _, err := range test.errs {
					if _, ok := (*ferrs)[err]; !ok {
						t.Fatalf("test[%d] - expected field '%s' to be in *webutil.Errors\n", i, err)
					}
				}
				continue
			}
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}
	}
}
