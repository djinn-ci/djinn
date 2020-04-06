package repo

import (
	"database/sql/driver"
	"fmt"
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/provider"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var repoCols = []string{
	"user_id",

}

func store(t *testing.T) (Store, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_StoreGet(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM provider_repos",
			[]query.Option{},
			sqlmock.NewRows(repoCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM provider_repos WHERE (user_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(repoCols),
			[]driver.Value{1},
			[]model.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM provider_repos WHERE (user_id = $1 AND provider_id = $2)",
			[]query.Option{},
			sqlmock.NewRows(repoCols),
			[]driver.Value{1, 1},
			[]model.Model{&user.User{ID: 1}, &provider.Provider{ID: 1}},
		},
	}

	for _, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Error(errors.Cause(err))
		}

		store.User = nil
		store.Provider = nil
	}
}

func Test_StoreAll(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM provider_repos",
			[]query.Option{},
			sqlmock.NewRows(repoCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM provider_repos WHERE (user_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(repoCols),
			[]driver.Value{1},
			[]model.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM provider_repos WHERE (user_id = $1 AND provider_id = $2)",
			[]query.Option{},
			sqlmock.NewRows(repoCols),
			[]driver.Value{1, 1},
			[]model.Model{&user.User{ID: 1}, &provider.Provider{ID: 1}},
		},
	}

	for _, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Error(errors.Cause(err))
		}

		store.User = nil
		store.Provider = nil
	}
}

func Test_StoreCreate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	r := &Repo{}

	id := int64(10)
	expected := fmt.Sprintf(insertFmt, table)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectPrepare(expected).ExpectQuery().WillReturnRows(rows)

	if err := store.Create(r); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if r.ID != id {
		t.Fatalf("repo id mismatch\n\texpected = '%d'\n\tactual   = '%d'\n", id, r.ID)
	}
}


func Test_StoreDelete(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	rr := []*Repo{
		&Repo{ID: 1},
		&Repo{ID: 2},
		&Repo{ID: 3},
	}

	expected := fmt.Sprintf(deleteFmt, table)

	mock.ExpectPrepare(expected).ExpectExec().WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(rr...); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
