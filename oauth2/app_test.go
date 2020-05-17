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

var appCols = []string{
	"user_id",
	"client_id",
	"client_secret",
	"name",
	"description",
	"home_uri",
	"redirect_uri",
}

func appStore(t *testing.T) (AppStore, sqlmock.Sqlmock, func() error) {
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
			[]model.Model{},
		},
		{
			"SELECT * FROM oauth_apps WHERE (user_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(appCols),
			[]driver.Value{1},
			[]model.Model{&user.User{ID: 1}},
		},
	}

	for _, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatal(errors.Cause(err))
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
			[]model.Model{},
		},
		{
			"SELECT * FROM oauth_apps WHERE (user_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(appCols),
			[]driver.Value{1},
			[]model.Model{&user.User{ID: 1}},
		},
	}

	for _, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatal(errors.Cause(err))
		}

		store.User = nil
	}
}

func Test_AppStoreCreate(t *testing.T) {
	store, mock, close_ := appStore(t)
	defer close_()

	a := &App{}

	id := int64(0)
	expected := fmt.Sprintf(insertFmt, appTable)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectPrepare(expected).ExpectQuery().WillReturnRows(rows)

	if err := store.Create(a); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if a.ID != id {
		t.Fatalf("app id mismatch\n\texpected = '%d'\n\tactual   = '%d'\n", id, a.ID)
	}
}

func Test_AppStoreUpdate(t *testing.T) {
	store, mock, close_ := appStore(t)
	defer close_()

	a := &App{ID: 10}

	expected := fmt.Sprintf(updateFmt, appTable)

	mock.ExpectPrepare(expected).ExpectExec().WillReturnResult(sqlmock.NewResult(a.ID, 1))

	if err := store.Update(a); err != nil {
		t.Fatal(errors.Cause(err))
	}
}

func Test_AppStoreDelete(t *testing.T) {
	store, mock, close_ := appStore(t)
	defer close_()

	aa := []*App{
		&App{ID: 1},
		&App{ID: 2},
		&App{ID: 3},
	}

	expected := fmt.Sprintf(deleteFmt, appTable)

	mock.ExpectPrepare(expected).ExpectExec().WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(aa...); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
