package build

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/database"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var jobCols = []string{
	"build_id",
	"stage_id",
	"name",
	"commands",
	"status",
	"output",
	"started_at",
	"finished_at",
}

func jobStore(t *testing.T) (*JobStore, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewJobStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_JobStoreAll(t *testing.T) {
	store, mock, close_ := jobStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM build_jobs",
			[]query.Option{},
			sqlmock.NewRows(jobCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM build_jobs WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(jobCols),
			[]driver.Value{10},
			[]database.Model{&Build{ID: 10}},
		},
		{
			"SELECT * FROM build_jobs WHERE (stage_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(jobCols),
			[]driver.Value{10},
			[]database.Model{&Stage{ID: 10}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.Build = nil
		store.Stage = nil
	}
}

func Test_JobStoreGet(t *testing.T) {
	store, mock, close_ := jobStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM build_jobs",
			[]query.Option{},
			sqlmock.NewRows(jobCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM build_jobs WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(jobCols),
			[]driver.Value{10},
			[]database.Model{&Build{ID: 10}},
		},
		{
			"SELECT * FROM build_jobs WHERE (stage_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(jobCols),
			[]driver.Value{10},
			[]database.Model{&Stage{ID: 10}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.Build = nil
		store.Stage = nil
	}
}
