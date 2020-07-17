package repo

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/provider"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

type testQuery struct {
	query  string
	opts   []query.Option
	rows   *sqlmock.Rows
	args   []driver.Value
	models []database.Model
}

var repoCols = []string{
	"user_id",
	"provider_id",
	"hook_id",
	"repo_id",
	"enabled",
}

func store(t *testing.T) (*Store, sqlmock.Sqlmock, func() error) {
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
			[]database.Model{},
		},
		{
			"SELECT * FROM provider_repos WHERE (user_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(repoCols),
			[]driver.Value{1},
			[]database.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM provider_repos WHERE (user_id = $1 AND provider_id = $2)",
			[]query.Option{},
			sqlmock.NewRows(repoCols),
			[]driver.Value{1, 1},
			[]database.Model{&user.User{ID: 1}, &provider.Provider{ID: 1}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
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
			[]database.Model{},
		},
		{
			"SELECT * FROM provider_repos WHERE (user_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(repoCols),
			[]driver.Value{1},
			[]database.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM provider_repos WHERE (user_id = $1 AND provider_id = $2)",
			[]query.Option{},
			sqlmock.NewRows(repoCols),
			[]driver.Value{1, 1},
			[]database.Model{&user.User{ID: 1}, &provider.Provider{ID: 1}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.User = nil
		store.Provider = nil
	}
}

func Test_StoreCreate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	mock.ExpectQuery(
		"^INSERT INTO provider_repos \\((.+)\\) VALUES \\((.+)\\) RETURNING id$",
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))

	if _, err := store.Create(143, 1897); err != nil {
		t.Errorf("unexpected Create error: %s\n", errors.Cause(err))
	}
}

func Test_StoreDelete(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	mock.ExpectExec(
		"^DELETE FROM provider_repos WHERE \\(id IN \\(\\$1, \\$2, \\$3\\)\\)$",
	).WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(1, 2, 3); err != nil {
		t.Errorf("unexpected Delete error: %s\n", errors.Cause(err))
	}
}
