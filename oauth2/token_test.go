package oauth2

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var tokenCols = []string{
	"user_id",
	"app_id",
	"name",
	"token",
	"scope",
}

func tokenStore(t *testing.T) (*TokenStore, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewTokenStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_TokenStoreGet(t *testing.T) {
	store, mock, close_ := tokenStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM oauth_tokens",
			[]query.Option{},
			sqlmock.NewRows(tokenCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM oauth_tokens WHERE (user_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(tokenCols),
			[]driver.Value{1},
			[]database.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM oauth_tokens WHERE (app_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(tokenCols),
			[]driver.Value{1},
			[]database.Model{&App{ID: 1}},
		},
		{
			"SELECT * FROM oauth_tokens WHERE (user_id = $1 AND app_id = $2)",
			[]query.Option{},
			sqlmock.NewRows(tokenCols),
			[]driver.Value{1, 1},
			[]database.Model{&App{ID: 1}, &user.User{ID: 1}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.App = nil
		store.User = nil
	}
}

func Test_TokenStoreCreate(t *testing.T) {
	store, mock, close_ := tokenStore(t)
	defer close_()

	mock.ExpectQuery(
		"^INSERT INTO oauth_tokens \\((.+)\\) VALUES \\((.+)\\) RETURNING id$",
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))

	sc, _ := UnmarshalScope("build:read,write,delete")

	if _, err := store.Create("build token", sc); err != nil {
		t.Errorf("unexpected Create error: %s\n", errors.Cause(err))
	}
}

func Test_TokenStoreUpdate(t *testing.T) {
	store, mock, close_ := tokenStore(t)
	defer close_()

	mock.ExpectExec(
		"UPDATE oauth_tokens SET name = \\$1, scope = \\$2 WHERE \\(id = \\$3\\)$",
	).WillReturnResult(sqlmock.NewResult(0, 1))

	sc, _ := UnmarshalScope("build:read,write,delete")

	if err := store.Update(10, "dev token", sc); err != nil {
		t.Errorf("unexpected Update error: %s\n", errors.Cause(err))
	}
}

func Test_TokenStoreDelete(t *testing.T) {
	store, mock, close_ := tokenStore(t)
	defer close_()

	mock.ExpectExec(
		"^DELETE FROM oauth_tokens WHERE \\(id IN \\(\\$1, \\$2, \\$3\\)\\)",
	).WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(1, 2, 3); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
