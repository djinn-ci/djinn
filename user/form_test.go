package user

import (
	"regexp"
	"testing"

	"djinn-ci.com/errors"

	"github.com/andrewpillar/webutil"
)

func Test_RegisterForm(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []struct {
		form        RegisterForm
		user        *User
		errs        []string
		shouldError bool
	}{
		{
			RegisterForm{
				Users:          store,
				Email:          "freemang@black-mesa.com",
				Username:       "freemang",
				Password:       "secret",
				VerifyPassword: "secret",
			},
			&User{},
			[]string{},
			false,
		},
		{
			RegisterForm{
				Users:          store,
				Email:          "freemang@black-mesa.com",
				Username:       "freemang",
				Password:       "secret",
				VerifyPassword: "secret",
			},
			&User{Email: "freemang@black-mesa.com"},
			[]string{"email"},
			true,
		},
		{
			RegisterForm{
				Users:          store,
				Email:          "freemang@black-mesa.com",
				Username:       "freemang",
				Password:       "secret",
				VerifyPassword: "secret",
			},
			&User{Username: "freemang"},
			[]string{"username"},
			true,
		},
		{
			RegisterForm{Users: store},
			&User{},
			[]string{"email", "username", "password", "verify_password"},
			true,
		},
		{
			RegisterForm{
				Users:          store,
				Email:          "freemang",
				Username:       "freemang",
				Password:       "secret",
				VerifyPassword: "secret",
			},
			&User{},
			[]string{"email"},
			true,
		},
		{
			RegisterForm{
				Users:          store,
				Email:          "freemang@black-mesa.com",
				Username:       "freemang",
				Password:       "secret",
				VerifyPassword: "",
			},
			&User{},
			[]string{"verify_password"},
			true,
		},
		{
			RegisterForm{
				Users:          store,
				Email:          "freemang@black-mesa.com",
				Username:       "freemang",
				Password:       "secret",
				VerifyPassword: "terces",
			},
			&User{},
			[]string{"password", "verify_password"},
			true,
		},
	}

	for i, test := range tests {
		rows := mock.NewRows([]string{"email", "username"}).AddRow(test.user.Email, test.user.Username)

		mock.ExpectQuery(
			regexp.QuoteMeta("SELECT * FROM users WHERE (email = $1 OR username = $2)"),
		).WithArgs(test.form.Email, test.form.Username).WillReturnRows(rows)

		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				if len(test.errs) == 0 {
					continue
				}

				ferrs, ok := err.(*webutil.Errors)

				if !ok {
					t.Fatalf("test[%d] - expected error to be *webutil.Errors, it was '%s'\n", i, errors.Cause(err))
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

func Test_LoginForm(t *testing.T) {
	tests := []struct {
		form        LoginForm
		errs        []string
		shouldError bool
	}{
		{
			LoginForm{Handle: "freemang", Password: "secret"},
			[]string{},
			false,
		},
		{
			LoginForm{Handle: "freemang"},
			[]string{"password"},
			true,
		},
		{
			LoginForm{Password: "secret"},
			[]string{"handle"},
			true,
		},
		{
			LoginForm{},
			[]string{"handle", "password"},
			true,
		},
	}

	for i, test := range tests {
		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				if len(test.errs) == 0 {
					continue
				}

				ferrs, ok := err.(*webutil.Errors)

				if !ok {
					t.Fatalf("test[%d] - expected error to be *webutil.Errors it was '%s'\n", i, errors.Cause(err))
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

func Test_EmailForm(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []struct {
		form        EmailForm
		other       *User
		errs        []string
		shouldQuery bool
		shouldError bool
	}{
		{
			EmailForm{
				Users: store,
				User: &User{
					Email:    "freemang@black-mesa.com",
					Password: bcryptPassword,
				},
				Email:          "gordon.freeman@black-mesa.com",
				VerifyPassword: "secret",
			},
			&User{},
			[]string{},
			true,
			false,
		},
		{
			EmailForm{
				Users: store,
				User: &User{
					Email:    "freemang@black-mesa.com",
					Password: bcryptPassword,
				},
				Email:          "freemang@black-mesa.com",
				VerifyPassword: "secret",
			},
			&User{},
			[]string{},
			false,
			false,
		},
		{
			EmailForm{
				Users: store,
				User: &User{
					Email:    "freemang@black-mesa.com",
					Password: bcryptPassword,
				},
				Email:          "gordon.freeman",
				VerifyPassword: "secret",
			},
			&User{},
			[]string{"email"},
			true,
			true,
		},
		{
			EmailForm{
				Users: store,
				User: &User{
					Email:    "freemang@black-mesa.com",
					Password: bcryptPassword,
				},
				Email:          "gordon.freeman@black-mesa.com",
				VerifyPassword: "tecres",
			},
			&User{},
			[]string{"email_verify_password"},
			true,
			true,
		},
		{
			EmailForm{
				Users: store,
				User: &User{
					Email:    "freemang@black-mesa.com",
					Password: bcryptPassword,
				},
				Email:          "breenw@black-mesa.com",
				VerifyPassword: "tecres",
			},
			&User{Email: "breenw@black-mesa.com"},
			[]string{"email"},
			true,
			true,
		},
	}

	for _, test := range tests {
		if test.shouldQuery {
			rows := mock.NewRows([]string{"email"}).AddRow(test.other.Email)

			mock.ExpectQuery(
				regexp.QuoteMeta("SELECT * FROM users WHERE (email = $1)"),
			).WithArgs(test.form.Email).WillReturnRows(rows)
		}

		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				if len(test.errs) == 0 {
					continue
				}

				ferrs, ok := err.(*webutil.Errors)

				if !ok {
					t.Fatalf("expected error to be *webutil.Errors, it was not\n%s\n", errors.Cause(err))
				}

				for _, err := range test.errs {
					if _, ok := (*ferrs)[err]; !ok {
						t.Fatalf("expected field '%s' to be in *webutil.Errors, it was not\n", err)
					}
				}
				continue
			}
			t.Fatal(errors.Cause(err))
		}
	}
}
