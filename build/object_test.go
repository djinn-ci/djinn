package build

import (
	"database/sql/driver"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/object"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var (
	objectCols = []string{
		"build_id",
		"object_id",
		"source",
		"name",
		"placed",
	}

	objectModel = &object.Object{
		ID:        1,
		UserID:    1,
		Hash:      "asemkcalb",
		Name:      "crowbar",
		Type:      "application/octet-stream",
		MD5:       []byte("34e927c45cf3ccebb09b006b00f4e02d"),
		SHA256:    []byte("be0c96b17d8481575fa72aee2fe75c42f269bfb893012fcaffb561583ba07827"),
		CreatedAt: time.Now(),
	}
)

func objectStore(t *testing.T) (ObjectStore, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewObjectStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_ObjectStoreAll(t *testing.T) {
	store, mock, close_ := objectStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM build_objects",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM build_objects WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{10},
			[]model.Model{&Build{ID: 10}},
		},
		{
			"SELECT * FROM build_objects WHERE (object_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{1},
			[]model.Model{objectModel},
		},
	}

	for _, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatal(errors.Cause(err))
		}

		store.Build = nil
		store.Object = nil
	}
}

func Test_ObjectStoreGet(t *testing.T) {
	store, mock, close_ := objectStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM build_objects",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM build_objects WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{10},
			[]model.Model{&Build{ID: 10}},
		},
		{
			"SELECT * FROM build_objects WHERE (object_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{1},
			[]model.Model{objectModel},
		},
	}

	for _, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatal(errors.Cause(err))
		}

		store.Build = nil
		store.Object = nil
	}
}

func Test_ObjectStoreCreate(t *testing.T) {
	store, mock, close_ := objectStore(t)
	defer close_()

	o := &Object{}

	id := int64(10)
	expected := fmt.Sprintf(insertFmt, objectTable)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectPrepare(expected).ExpectQuery().WillReturnRows(rows)

	if err := store.Create(o); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if o.ID != id {
		t.Fatalf("object id mismatch\n\texpected = '%d'\n\tactual   = '%d'\n", id, o.ID)
	}
}

func Test_ObjectStoreUpdate(t *testing.T) {
	store, mock, close_ := objectStore(t)
	defer close_()

	o := &Object{}

	expected := fmt.Sprintf(updateFmt, objectTable)

	mock.ExpectPrepare(expected).ExpectExec().WillReturnResult(sqlmock.NewResult(o.ID, 1))

	if err := store.Update(o); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
