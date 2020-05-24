package namespace

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/user"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

func userStore(t *testing.T) (*user.Store, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return user.NewStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_FormValidate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []struct{
		form        Form
		errs        []string
		query       string
		args        []driver.Value
		shouldError bool
	}{
		{
			Form{
				Namespaces:  store,
				Name:        "",
				Description: "",
				Visibility:  Private,
			},
			[]string{"name"},
			"SELECT * FROM namespaces WHERE (path = $1)",
			[]driver.Value{""},
			true,
		},
		{
			Form{
				Namespaces:  store,
				Name:        "bl",
				Description: "",
				Visibility:  Private,
			},
			[]string{"name"},
			"SELECT * FROM namespaces WHERE (path = $1)",
			[]driver.Value{"bl"},
			true,
		},
		{
			Form{
				Namespaces:  store,
				Name:        "",
				Description: "black-mesa gordon freeman g-man alyx vance eli vance city 17 combine striders antlion lost coast blueshift adrian shephard barney calhoun opposing force borealis aperture ravenholm wallace breen resonance cascade vortigaunt seven hour war the citadel crowbar",
				Visibility:  Private,
			},
			[]string{"name", "description"},
			"SELECT * FROM namespaces WHERE (path = $1)",
			[]driver.Value{""},
			true,
		},
		{
			Form{
				Namespaces: store,
				Name:       "black-mesa",
				Visibility: Private,
			},
			[]string{"name"},
			"SELECT * FROM namespaces WHERE (path = $1)",
			[]driver.Value{"black-mesa"},
			true,
		},
		{
			Form{
				Namespaces: store,
				Namespace:  &Namespace{Name: "blackmesa"},
				Name:       "blackmesa",
				Visibility:  Private,
			},
			[]string{""},
			"",
			[]driver.Value{},
			false,
		},
	}

	for i, test := range tests {
		if test.query != "" {
			mock.ExpectQuery(
				regexp.QuoteMeta(test.query),
			).WithArgs(test.args...).WillReturnRows(sqlmock.NewRows(namespaceCols))
		}

		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				if len(test.errs) == 0 {
					continue
				}

				ferrs, ok := err.(form.Errors)

				if !ok {
					t.Fatalf("test[%d] - expected error to be form.Errors, it was %s\n", i, errors.Cause(err))
				}

				for _, err := range test.errs {
					if _, ok := ferrs[err]; !ok {
						t.Fatalf("test[%d] - expected '%s' in form.Errors\n", i, err)
					}
				}
				continue
			}
			t.Fatalf("test[%d] - %s\n", i ,errors.Cause(err))
		}
	}
}

func Test_InviteFormValidate(t *testing.T) {
	userStore, userMock, userClose := userStore(t)
	inviteStore, inviteMock, inviteClose := inviteStore(t)
	collaboratorStore, collabMock, collabClose := collaboratorStore(t)

	defer userClose()
	defer inviteClose()
	defer collabClose()

	tests := []struct{
		form        InviteForm
		userq       string
		inviteq     string
		collabq     string
		args        []driver.Value
		userRows    []string
		errs        []string
		shouldError bool
	}{
		{
			InviteForm{
				Collaborators: collaboratorStore,
				Invites:       inviteStore,
				Users:         userStore,
				Inviter:       &user.User{},
			},
			"SELECT * FROM users WHERE (email = $1 OR username = $2)",
			"SELECT * FROM namespace_invites WHERE (invitee_id = (SELECT id FROM users WHERE (email = $1 OR username = $2)))",
			"SELECT * FROM namespace_collaborators WHERE (user_id = (SELECT id FROM users WHERE (email = $1 OR username = $2)))",
			[]driver.Value{"", ""},
			[]string{"email", "username", "password", "created_at"},
			[]string{"handle"},
			true,
		},
		{
			InviteForm{
				Collaborators: collaboratorStore,
				Invites:       inviteStore,
				Users:         userStore,
				Inviter:       &user.User{
					Username: "alyx.vance",
				},
				Handle:        "alyx.vance",
			},
			"SELECT * FROM users WHERE (email = $1 OR username = $2)",
			"SELECT * FROM namespace_invites WHERE (invitee_id = (SELECT id FROM users WHERE (email = $1 OR username = $2)))",
			"SELECT * FROM namespace_collaborators WHERE (user_id = (SELECT id FROM users WHERE (email = $1 OR username = $2)))",
			[]driver.Value{"alyx.vance", "alyx.vance"},
			[]string{"email", "username", "password", "created_at"},
			[]string{"handle"},
			true,
		},
	}

	for i, test := range tests {
		userMock.ExpectQuery(
			regexp.QuoteMeta(test.userq),
		).WithArgs(test.args...).WillReturnRows(sqlmock.NewRows(test.userRows))

		inviteMock.ExpectQuery(
			regexp.QuoteMeta(test.inviteq),
		).WithArgs(test.args...).WillReturnRows(sqlmock.NewRows(inviteCols))

		collabMock.ExpectQuery(
			regexp.QuoteMeta(test.collabq),
		).WithArgs(test.args...).WillReturnRows(sqlmock.NewRows(collaboratorCols))

		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				continue
			}
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}
	}
}
