package image

import (
	"database/sql/driver"
	"fmt"
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var imageCols = []string{
	"user_id",
	"namespace_id",
	"driver",
	"hash",
	"name",
	"created_at",
}

func store(t *testing.T) (*Store, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_StoreAll(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM images",
			[]query.Option{},
			sqlmock.NewRows(imageCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM images WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(imageCols),
			[]driver.Value{1, 1, 1},
			[]model.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM images WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(imageCols),
			[]driver.Value{1},
			[]model.Model{&namespace.Namespace{ID: 1}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.User = nil
		store.Namespace = nil
	}
}

func Test_StoreGet(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM images",
			[]query.Option{},
			sqlmock.NewRows(imageCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM images WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(imageCols),
			[]driver.Value{1, 1, 1},
			[]model.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM images WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(imageCols),
			[]driver.Value{1},
			[]model.Model{&namespace.Namespace{ID: 1}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.User = nil
		store.Namespace = nil
	}
}

func Test_StoreCreate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	i := &Image{}

	id := int64(10)
	expected := fmt.Sprintf(insertFmt, table)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectPrepare(expected).ExpectQuery().WillReturnRows(rows)

	if err := store.Create(i); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if i.ID != id {
		t.Fatalf("image id mismatch\n\texpected = '%d'\n\tactual   = '%d'\n", id, i.ID)
	}
}

func Test_StoreDelete(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	ii := []*Image{
		&Image{ID: 1},
		&Image{ID: 2},
		&Image{ID: 3},
	}

	expected := fmt.Sprintf(deleteFmt, table)

	mock.ExpectPrepare(expected).ExpectExec().WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(ii...); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
