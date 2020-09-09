package build

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var driverCols = []string{
	"build_id",
	"type",
	"config",
}

func driverStore(t *testing.T) (*DriverStore, sqlmock.Sqlmock, func() error) {
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
			[]database.Model{},
		},
		{
			"SELECT * FROM build_drivers WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(driverCols),
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

func Test_DriverStoreGet(t *testing.T) {
	store, mock, close_ := driverStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM build_drivers",
			[]query.Option{},
			sqlmock.NewRows(driverCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM build_drivers WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(driverCols),
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

func Test_DriverStoreCreate(t *testing.T) {
	store, mock, close_ := driverStore(t)
	defer close_()

	mock.ExpectQuery(
		"^INSERT INTO build_drivers \\([\\w+, ]+\\) VALUES \\([\\$\\d+, ]+\\) RETURNING id$",
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))

	if _, err := store.Create(map[string]string{}); err == nil {
		t.Errorf("expected Create to error, it did not\n")
	}

	if _, err := store.Create(map[string]string{"type": "qemu"}); err != nil {
		t.Errorf("unexpected Create error: %s\n", errors.Cause(err))
	}
}
