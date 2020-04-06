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

var codeCols = []string{
	"user_id",
	"app_id",
	"code",
	"scope",
	"expires_at",
}

func codeStore(t *testing.T) (CodeStore, sqlmock.Sqlmock, func() error) {
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
			[]model.Model{},
		},
		{
			"SELECT * FROM oauth_codes WHERE (user_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(codeCols),
			[]driver.Value{1},
			[]model.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM oauth_codes WHERE (user_id = $1 AND app_id = $2)",
			[]query.Option{},
			sqlmock.NewRows(codeCols),
			[]driver.Value{1, 1},
			[]model.Model{&user.User{ID: 1}, &App{ID: 1}},
		},
	}

	for _, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatal(errors.Cause(err))
		}

		store.User = nil
		store.App = nil
	}
}

func Test_CodeStoreCreate(t *testing.T) {
	store, mock, close_ := codeStore(t)
	defer close_()

	c := &Code{}

	id := int64(0)
	expected := fmt.Sprintf(insertFmt, codeTable)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectPrepare(expected).ExpectQuery().WillReturnRows(rows)

	if err := store.Create(c); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if c.ID != id {
		t.Fatalf("code id mismatch\n\texpected = '%d'\n\tactual   = '%d'\n", id, c.ID)
	}
}

func Test_CodeStoreDelete(t *testing.T) {
	store, mock, close_ := codeStore(t)
	defer close_()

	cc := []*Code{
		&Code{ID: 1},
		&Code{ID: 2},
		&Code{ID: 3},
	}

	expected := fmt.Sprintf(deleteFmt, codeTable)

	mock.ExpectPrepare(expected).ExpectExec().WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(cc...); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
