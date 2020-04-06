package key

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

var keyCols = []string{
	"user_id",
	"namespace_id",
	"name",
	"key",
	"config",
}

func store(t *testing.T) (Store, sqlmock.Sqlmock, func() error) {
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
			"SELECT * FROM keys",
			[]query.Option{},
			sqlmock.NewRows(keyCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM keys WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(keyCols),
			[]driver.Value{1, 1, 1},
			[]model.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM keys WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(keyCols),
			[]driver.Value{1},
			[]model.Model{&namespace.Namespace{ID: 1}},
		},
	}

	for _, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatal(errors.Cause(err))
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
			"SELECT * FROM keys",
			[]query.Option{},
			sqlmock.NewRows(keyCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM keys WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(keyCols),
			[]driver.Value{1, 1, 1},
			[]model.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM keys WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(keyCols),
			[]driver.Value{1},
			[]model.Model{&namespace.Namespace{ID: 1}},
		},
	}

	for _, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatal(errors.Cause(err))
		}

		store.User = nil
		store.Namespace = nil
	}
}

func Test_StoreCreate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	k := &Key{}

	id := int64(10)
	expected := fmt.Sprintf(insertFmt, table)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectPrepare(expected).ExpectQuery().WillReturnRows(rows)

	if err := store.Create(k); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if k.ID != id {
		t.Fatalf("key id mismatch\n\texpected = '%d'\n\tactual   = '%d'\n", id, k.ID)
	}
}

func Test_StoreUpdate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	k := &Key{ID: 10}

	expected := fmt.Sprintf(updateFmt, table)

	mock.ExpectPrepare(expected).ExpectExec().WillReturnResult(sqlmock.NewResult(k.ID, 1))

	if err := store.Update(k); err != nil {
		t.Fatal(errors.Cause(err))
	}
}

func Test_StoreDelete(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	kk := []*Key{
		&Key{ID: 1},
		&Key{ID: 2},
		&Key{ID: 3},
	}

	expected := fmt.Sprintf(deleteFmt, table)

	mock.ExpectPrepare(expected).ExpectExec().WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(kk...); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
