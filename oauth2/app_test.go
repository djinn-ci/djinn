package oauth2

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var appCols = []string{
	"user_id",
	"client_id",
	"client_secret",
	"name",
	"description",
	"home_uri",
	"redirect_uri",
}

func appStore(t *testing.T) (*AppStore, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewAppStore(sqlx.NewDb(db, "sqlx")), mock, db.Close
}

func Test_AppStoreAll(t *testing.T) {
	store, mock, close_ := appStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM oauth_apps",
			[]query.Option{},
			sqlmock.NewRows(appCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM oauth_apps WHERE (user_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(appCols),
			[]driver.Value{1},
			[]database.Model{&user.User{ID: 1}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.User = nil
	}
}

func Test_AppStoreGet(t *testing.T) {
	store, mock, close_ := appStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM oauth_apps",
			[]query.Option{},
			sqlmock.NewRows(appCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM oauth_apps WHERE (user_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(appCols),
			[]driver.Value{1},
			[]database.Model{&user.User{ID: 1}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.User = nil
	}
}

func Test_AppStoreCreate(t *testing.T) {
	store, mock, close_ := appStore(t)
	defer close_()

	block, err := crypto.NewBlock([]byte("some-supersecret"), []byte("some-salt"))

	if err != nil {
		t.Fatal(errors.Cause(err))
	}

	store.block = block

	mock.ExpectQuery(
		"^INSERT INTO oauth_apps \\((.+)\\) VALUES \\((.+)\\) RETURNING id$",
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))

	if _, err := store.Create("my app", "", "example.com", "example.com/oauth"); err != nil {
		t.Errorf("unexpected Create error: %s\n", errors.Cause(err))
	}
}

func Test_AppStoreUpdate(t *testing.T) {
	store, mock, close_ := appStore(t)
	defer close_()

	mock.ExpectExec(
		"^UPDATE oauth_apps SET name = \\$1, description = \\$2, home_uri = \\$3, redirect_uri = \\$4 WHERE \\(id = \\$5\\)$",
	).WithArgs("my app", "", "example.com", "example.com/oauth", 10).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.Update(10, "my app", "", "example.com", "example.com/oauth"); err != nil {
		t.Errorf("unexpected Update error: %s\n", errors.Cause(err))
	}
}

func Test_AppStoreDelete(t *testing.T) {
	store, mock, close_ := appStore(t)
	defer close_()

	mock.ExpectExec(
		"^DELETE FROM oauth_apps WHERE \\(id IN \\(\\$1, \\$2, \\$3\\)\\)$",
	).WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(1, 2, 3); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
