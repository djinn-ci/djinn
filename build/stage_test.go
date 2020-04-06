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

var stageCols = []string{
	"build_id",
	"name",
	"can_fail",
	"status",
	"started_at",
	"finished_at",
}

func stageStore(t *testing.T) (StageStore, sqlmock.Sqlmock, func() error) {
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
			[]model.Model{},
		},
		{
			"SELECT * FROM build_stages WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(stageCols),
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

func Test_StageStoreCreate(t *testing.T) {
	store, mock, close_ := stageStore(t)
	defer close_()

	s := &Stage{}

	id := int64(10)
	expected := fmt.Sprintf(insertFmt, stageTable)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectPrepare(expected).ExpectQuery().WillReturnRows(rows)

	if err := store.Create(s); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if s.ID != id {
		t.Fatalf("stage id mismatch\n\texpected = '%d'\n\tactual   = '%d'\n", id, s.ID)
	}
}
