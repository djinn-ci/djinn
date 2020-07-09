package build

import (
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/database"
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

func objectStore(t *testing.T) (*ObjectStore, sqlmock.Sqlmock, func() error) {
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
			[]database.Model{},
		},
		{
			"SELECT * FROM build_objects WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{10},
			[]database.Model{&Build{ID: 10}},
		},
		{
			"SELECT * FROM build_objects WHERE (object_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{1},
			[]database.Model{objectModel},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
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
			[]database.Model{},
		},
		{
			"SELECT * FROM build_objects WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{10},
			[]database.Model{&Build{ID: 10}},
		},
		{
			"SELECT * FROM build_objects WHERE (object_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{1},
			[]database.Model{objectModel},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.Build = nil
		store.Object = nil
	}
}

func Test_ObjectStoreCreate(t *testing.T) {
	store, mock, close_ := objectStore(t)
	defer close_()

	mock.ExpectQuery(
		"^INSERT INTO build_objects \\([\\w+, ]+\\) VALUES \\([\\$\\d+, ]+\\) RETURNING id$",
	).WithArgs().WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(14))

	if _, err := store.Create(10, "random-number", "rand"); err != nil {
		t.Errorf("unexpected Create error: %s\n", errors.Cause(err))
	}
}
