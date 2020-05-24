package build

import (
	"database/sql/driver"
	"fmt"
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

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
			[]model.Model{},
		},
		{
			"SELECT * FROM build_tags WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(tagCols),
			[]driver.Value{10},
			[]model.Model{&Build{ID: 10}},
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
			[]model.Model{},
		},
		{
			"SELECT * FROM build_tags WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(tagCols),
			[]driver.Value{10},
			[]model.Model{&Build{ID: 10}},
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

	tag := &Tag{}

	id := int64(10)
	expected := fmt.Sprintf(insertFmt, tagTable)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectPrepare(expected).ExpectQuery().WillReturnRows(rows)

	if err := store.Create(tag); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if tag.ID != id {
		t.Fatalf("stage id mismatch\n\texpected = '%d'\n\tactual   = '%d'\n", id, tag.ID)
	}
}

func Test_TagStoreDelete(t *testing.T) {
	store, mock, close_ := tagStore(t)
	defer close_()

	tt := []*Tag{
		&Tag{ID: 1},
		&Tag{ID: 2},
		&Tag{ID: 3},
	}

	expected := fmt.Sprintf(deleteFmt, tagTable)

	mock.ExpectPrepare(expected).ExpectExec().WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(tt...); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
