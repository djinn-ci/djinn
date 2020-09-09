package build

import (
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/variable"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var (
	variableCols = []string{
		"build_id",
		"variable_id",
		"key",
		"value",
	}

	variableModel = &variable.Variable{
		ID:        1,
		UserID:    1,
		Key:       "city",
		Value:     "17",
		CreatedAt: time.Now(),
	}
)

func variableStore(t *testing.T) (*VariableStore, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewVariableStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_VariableStoreAll(t *testing.T) {
	store, mock, close_ := variableStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM build_variables",
			[]query.Option{},
			sqlmock.NewRows(variableCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM build_variables WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(variableCols),
			[]driver.Value{10},
			[]database.Model{&Build{ID: 10}},
		},
		{
			"SELECT * FROM build_variables WHERE (variable_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(variableCols),
			[]driver.Value{1},
			[]database.Model{variableModel},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.Build = nil
		store.Variable = nil
	}
}
