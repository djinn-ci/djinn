package database

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

func store(t *testing.T) (Store, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return Store{
		DB: sqlx.NewDb(db, "sqlmock"),
	}, mock, db.Close
}

func Test_CompareKeys(t *testing.T) {
	tests := []struct{
		a           interface{}
		b           interface{}
		shouldMatch bool
	}{
		{int64(10), sql.NullInt64{Int64: 10}, true},
		{int64(10), int64(10), true},
		{sql.NullInt64{Int64: 10}, sql.NullInt64{Int64: 10}, true},
		{int64(10), sql.NullInt64{Int64: 5}, false},
		{int64(10), int64(5), false},
		{sql.NullInt64{Int64: 10}, sql.NullInt64{Int64: 5}, false},
	}

	for i, test := range tests {
		if !CompareKeys(test.a, test.b) {
			if !test.shouldMatch {
				continue
			}
			t.Errorf("test[%d] - expected %v to equal %v\n", i, test.a, test.b)
		}
	}
}
