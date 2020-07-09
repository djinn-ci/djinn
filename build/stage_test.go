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

var stageCols = []string{
	"build_id",
	"name",
	"can_fail",
	"status",
	"started_at",
	"finished_at",
}

func stageStore(t *testing.T) (*StageStore, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewStageStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_StageStoreAll(t *testing.T) {
	store, mock, close_ := stageStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM build_stages",
			[]query.Option{},
			sqlmock.NewRows(stageCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM build_stages WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(stageCols),
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
