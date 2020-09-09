package variable

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/user"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

func namespaceStore(t *testing.T) (*namespace.Store, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return namespace.NewStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_FormValidate(t *testing.T) {
	variableStore, variableMock, variableClose := store(t)
	defer variableClose()

	namespaceStore, namespaceMock, namespaceClose := namespaceStore(t)
	defer namespaceClose()

	tests := []struct {
		form        Form
		errs        []string
		shouldError bool
	}{
		{
			Form{
				Variables: variableStore,
				Key:       "PGADDR",
				Value:     "postgres://thrall:secret@localhost:5432/thrall",
			},
			[]string{},
			false,
		},
		{
			Form{
				Variables: variableStore,
				Key:       "0PGADDR",
				Value:     "postgres://thrall:secret@localhost:5432/thrall",
			},
			[]string{"key"},
			true,
		},
		{
			Form{
				Variables: variableStore,
				Key:       "0PGADDR",
			},
			[]string{"value"},
			true,
		},
		{
			Form{
				Variables: variableStore,
				Key:       "0PGADDR",
			},
			[]string{"key", "value"},
			true,
		},
		{
			Form{
				Resource: namespace.Resource{
					User:       &user.User{ID: 10},
					Namespaces: namespaceStore,
					Namespace:  "aperture",
				},
				Variables: variableStore,
				Key:       "PGADDR",
				Value:     "postgres://thrall:secret@localhost:5432/thrall",
			},
			[]string{},
			false,
		},
	}

	for i, test := range tests {
		uniqueQuery := "SELECT * FROM variables WHERE (key = $1)"
		uniqueArgs := []driver.Value{test.form.Key}

		if test.form.Namespace != "" {
			var (
				collabId    int64 = 13
				namespaceId int64 = 1
				userId      int64 = 10
			)

			uniqueQuery = "SELECT * FROM variables WHERE (namespace_id = $1 AND key = $2)"
			uniqueArgs = []driver.Value{namespaceId, test.form.Key}

			namespaceMock.ExpectQuery(
				regexp.QuoteMeta("SELECT * FROM namespaces WHERE (path = $1)"),
			).WithArgs(test.form.Namespace).WillReturnRows(
				sqlmock.NewRows([]string{"id", "root_id"}).AddRow(namespaceId, namespaceId),
			)

			namespaceMock.ExpectQuery(
				regexp.QuoteMeta("SELECT * FROM namespace_collaborators WHERE (namespace_id = $1)"),
			).WithArgs(namespaceId).WillReturnRows(
				sqlmock.NewRows([]string{"id", "user_id", "namespace_id"}).AddRow(collabId, userId, namespaceId),
			)
		}

		variableMock.ExpectQuery(
			regexp.QuoteMeta(uniqueQuery),
		).WithArgs(uniqueArgs...).WillReturnRows(sqlmock.NewRows(variableCols))

		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				cause := errors.Cause(err)

				ferrs, ok := cause.(form.Errors)

				if !ok {
					t.Fatalf("test[%d] - expected error to be form.Errors, is was '%s'\n", i, cause)
				}

				for _, err := range test.errs {
					if _, ok := ferrs[err]; !ok {
						t.Errorf("test[%d] - expected '%s' to be in form.Errors\n", i, err)
					}
				}
				continue
			}
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}
	}
}
