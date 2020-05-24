package oauth2

import (
	"database/sql/driver"
	"fmt"
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
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
			[]model.Model{},
		},
		{
			"SELECT * FROM oauth_tokens WHERE (user_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(tokenCols),
			[]driver.Value{1,},
			[]model.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM oauth_tokens WHERE (app_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(tokenCols),
			[]driver.Value{1},
			[]model.Model{&App{ID: 1}},
		},
		{
			"SELECT * FROM oauth_tokens WHERE (user_id = $1 AND app_id = $2)",
			[]query.Option{},
			sqlmock.NewRows(tokenCols),
			[]driver.Value{1, 1},
			[]model.Model{&App{ID: 1}, &user.User{ID: 1}},
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

	tok := &Token{}

	id := int64(0)
	expected := fmt.Sprintf(insertFmt, tokenTable)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectPrepare(expected).ExpectQuery().WillReturnRows(rows)

	if err := store.Create(tok); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if tok.ID != id {
		t.Fatalf("token id mismatch\n\texpected = '%d'\n\tactual   = '%d'\n", id, tok.ID)
	}
}

func Test_TokenStoreUpdate(t *testing.T) {
	store, mock, close_ := tokenStore(t)
	defer close_()

	tok := &Token{ID: 10}

	expected := fmt.Sprintf(updateFmt, tokenTable)

	mock.ExpectPrepare(expected).ExpectExec().WillReturnResult(sqlmock.NewResult(tok.ID, 1))

	if err := store.Update(tok); err != nil {
		t.Fatal(errors.Cause(err))
	}
}

func Test_TokenStoreDelete(t *testing.T) {
	store, mock, close_ := tokenStore(t)
	defer close_()

	tt := []*Token{
		&Token{ID: 1},
		&Token{ID: 2},
		&Token{ID: 3},
	}

	expected := fmt.Sprintf(deleteFmt, tokenTable)

	mock.ExpectPrepare(expected).ExpectExec().WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(tt...); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
