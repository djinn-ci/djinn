package oauth2

import (
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"

	"github.com/DATA-DOG/go-sqlmock"
)

func Test_AppForm(t *testing.T) {
	store, mock, close_ := appStore(t)
	defer close_()

	tests := []struct{
		form        AppForm
		errs        []string
		shouldError bool
	}{
		{
			AppForm{
				Apps:        store,
				Name:        "hl2",
				HomepageURI: "https://example.com",
				RedirectURI: "https://example.com/oauth/thrall",
			},
			[]string{},
			false,
		},
		{
			AppForm{
				Apps:        store,
				Name:        "hl2",
				HomepageURI: "https://example.com",
				RedirectURI: "https://example.com/oauth/thrall",
			},
			[]string{},
			false,
		},
		{
			AppForm{
				Apps:        store,
				HomepageURI: "https://example.com",
				RedirectURI: "https://example.com/oauth/thrall",
			},
			[]string{"name"},
			true,
		},
		{
			AppForm{
				Apps:        store,
				HomepageURI: "https://example.com",
				RedirectURI: "https://example.com/oauth/thrall",
			},
			[]string{"name"},
			true,
		},
		{
			AppForm{
				Apps:        store,
				RedirectURI: "https://example.com/oauth/thrall",
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

	for _, test := range tests {
		mock.ExpectQuery(
			regexp.QuoteMeta("SELECT * FROM oauth_apps WHERE (name = $1)"),
		).WithArgs(test.form.Name).WillReturnRows(sqlmock.NewRows(appCols))

		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				cause := errors.Cause(err)

				ferrs, ok := cause.(form.Errors)

				if !ok {
					t.Fatalf("expected error to be form.Errors, is was '%s'\n", cause)
				}

				for _, err := range test.errs {
					if _, ok := ferrs[err]; !ok {
						t.Fatalf("expected field '%s' to be in form.Errors, it was not\n", err)
					}
				}
				continue
			}
			t.Fatal(errors.Cause(err))
		}
	}
}

func Test_AuthorizeForm(t *testing.T) {
	tests := []struct{
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

	for _, test := range tests {
		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				cause := errors.Cause(err)

				ferrs, ok := cause.(form.Errors)

				if !ok {
					t.Fatalf("expected error to be form.Errors, is was '%s'\n", cause)
				}

				for _, err := range test.errs {
					if _, ok := ferrs[err]; !ok {
						t.Fatalf("expected field '%s' to be in form.Errors, it was not\n", err)
					}
				}
				continue
			}
			t.Fatal(errors.Cause(err))
		}
	}
}

func Test_TokenForm(t *testing.T) {
	store, mock, close_ := tokenStore(t)
	defer close_()

	tests := []struct{
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

	for _, test := range tests {
		mock.ExpectQuery(
			regexp.QuoteMeta("SELECT * FROM oauth_tokens WHERE (name = $1)"),
		).WithArgs(test.form.Name).WillReturnRows(sqlmock.NewRows(tokenCols))

		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				cause := errors.Cause(err)

				ferrs, ok := cause.(form.Errors)

				if !ok {
					t.Fatalf("expected error to be form.Errors, is was '%s'\n", cause)
				}

				for _, err := range test.errs {
					if _, ok := ferrs[err]; !ok {
						t.Fatalf("expected field '%s' to be in form.Errors, it was not\n", err)
					}
				}
				continue
			}
			t.Fatal(errors.Cause(err))
		}
	}
}