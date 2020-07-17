package build

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var tagCols = []string{
	"user_id",
	"build_id",
	"name",
}

func tagStore(t *testing.T) (*TagStore, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewTagStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_TagStoreAll(t *testing.T) {
	store, mock, close_ := tagStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM build_tags",
			[]query.Option{},
			sqlmock.NewRows(tagCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM build_tags WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(tagCols),
			[]driver.Value{10},
			[]database.Model{&Build{ID: 10}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.Build = nil
	}
}

func Test_TagStoreGet(t *testing.T) {
	store, mock, close_ := tagStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM build_tags",
			[]query.Option{},
			sqlmock.NewRows(tagCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM build_tags WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(tagCols),
			[]driver.Value{10},
			[]database.Model{&Build{ID: 10}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.Build = nil
	}
}

func Test_TagStoreCreate(t *testing.T) {
	store, mock, close_ := tagStore(t)
	defer close_()

	tests := []struct {
		buildId int64
		userId  int64
		names   []string
		queries []string
		rows    []*sqlmock.Rows
	}{
		{
			1,
			2,
			[]string{"tag1", "tag2", "tag3"},
			[]string{
				"^INSERT INTO build_tags (.+) VALUES (.+)$",
				"^INSERT INTO build_tags (.+) VALUES (.+)$",
				"^INSERT INTO build_tags (.+) VALUES (.+)$",
			},
			[]*sqlmock.Rows{
				sqlmock.NewRows([]string{"*"}).AddRow(10),
				sqlmock.NewRows([]string{"*"}).AddRow(11),
				sqlmock.NewRows([]string{"*"}).AddRow(12),
			},
		},
	}

	for i, test := range tests {
		for j, q := range test.queries {
			mock.ExpectQuery(q).WillReturnRows(test.rows[j])
		}

		if _, err := store.Create(test.userId, test.names...); err != nil {
			t.Errorf("tests[%d] - unexpected Create error: %s\n", i, errors.Cause(err))
		}
	}
}

func Test_TagStoreDelete(t *testing.T) {
	store, mock, close_ := tagStore(t)
	defer close_()

	mock.ExpectExec(
		"^DELETE FROM build_tags WHERE \\(id IN \\(\\$1, \\$2, \\$3\\)\\)$",
	).WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(1, 2, 3); err != nil {
		t.Errorf("unexpected Delete error: %s\n", errors.Cause(err))
	}
}
