package build

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"

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

	mock.ExpectQuery(
		"^SELECT name FROM build_tags WHERE (.+)$",
	).WillReturnRows(
		sqlmock.NewRows([]string{"name"}),
	)

	mock.ExpectQuery(
		"^INSERT INTO build_tags (.+) VALUES (.+)$",
	).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(1),
	)

	store.Bind(&Build{ID: 10})

	tags, err := store.Create(3, "tag1")

	if err != nil {
		t.Fatal(err)
	}

	tag := tags[0]

	if tag.UserID != 3 {
		t.Fatalf("unexpected UserID, expected=%d, got=%d\n", 3, tag.UserID)
	}
	if tag.BuildID != 10 {
		t.Fatalf("unexpected BuildID, expected=%d, got=%d\n", 10, tag.BuildID)
	}
	if tag.Name != "tag1" {
		t.Fatalf("unexpected Name, expected=%q, got=%q\n", "tag1", tag.Name)
	}
}

func Test_TagStoreDelete(t *testing.T) {
	store, mock, close_ := tagStore(t)
	defer close_()

	mock.ExpectExec(
		"^DELETE FROM build_tags WHERE \\(build_id = \\$1 AND name = \\$2\\)$",
	).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.Delete(1, "tag1"); err != nil {
		t.Errorf("unexpected Delete error: %s\n", errors.Cause(err))
	}
}
