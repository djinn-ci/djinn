package oauth2

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var codeCols = []string{
	"user_id",
	"app_id",
	"code",
	"scope",
	"expires_at",
}

func codeStore(t *testing.T) (*CodeStore, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewCodeStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_CodeStoreGet(t *testing.T) {
	store, mock, close_ := codeStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM oauth_codes",
			[]query.Option{},
			sqlmock.NewRows(codeCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM oauth_codes WHERE (user_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(codeCols),
			[]driver.Value{1},
			[]database.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM oauth_codes WHERE (user_id = $1 AND app_id = $2)",
			[]query.Option{},
			sqlmock.NewRows(codeCols),
			[]driver.Value{1, 1},
			[]database.Model{&user.User{ID: 1}, &App{ID: 1}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.User = nil
		store.App = nil
	}
}

func Test_CodeStoreCreate(t *testing.T) {
	store, mock, close_ := codeStore(t)
	defer close_()

	mock.ExpectQuery(
		"^INSERT INTO oauth_codes \\((.+)\\) VALUES \\((.+)\\) RETURNING id$",
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))

	sc, _ := UnmarshalScope("build:read")

	if _, err := store.Create(sc); err != nil {
		t.Errorf("unexpected Create error: %s\n", errors.Cause(err))
	}
}

func Test_CodeStoreDelete(t *testing.T) {
	store, mock, close_ := codeStore(t)
	defer close_()

	mock.ExpectExec(
		"^DELETE FROM oauth_codes WHERE \\(id IN \\(\\$1, \\$2, \\$3\\)\\)$",
	).WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(1, 2, 3); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
