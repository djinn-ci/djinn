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

var driverCols = []string{
	"build_id",
	"type",
	"config",
}

func driverStore(t *testing.T) (DriverStore, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewDriverStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_DriverStoreAll(t *testing.T) {
	store, mock, close_ := driverStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM build_drivers",
			[]query.Option{},
			sqlmock.NewRows(driverCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM build_drivers WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(driverCols),
			[]driver.Value{10},
			[]model.Model{&Build{ID: 10}},
		},
	}

	for _, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatal(errors.Cause(err))
		}

		store.Build = nil
	}
}

func Test_DriverStoreGet(t *testing.T) {
	store, mock, close_ := driverStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM build_drivers",
			[]query.Option{},
			sqlmock.NewRows(driverCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM build_drivers WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(driverCols),
			[]driver.Value{10},
			[]model.Model{&Build{ID: 10}},
		},
	}

	for _, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatal(errors.Cause(err))
		}

		store.Build = nil
	}
}

func Test_DriverStoreCreate(t *testing.T) {
	store, mock, close_ := driverStore(t)
	defer close_()

	d := &Driver{}

	id := int64(10)
	expected := fmt.Sprintf(insertFmt, driverTable)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectPrepare(expected).ExpectQuery().WillReturnRows(rows)

	if err := store.Create(d); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if d.ID != id {
		t.Fatalf("driver id mismatch\n\texpected = '%d'\n\tactual   = '%d'\n", id, d.ID)
	}
}
