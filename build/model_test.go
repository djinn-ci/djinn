package build

import (
	"database/sql/driver"

	"github.com/andrewpillar/djinn/database"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"
)

type testQuery struct {
	query  string
	opts   []query.Option
	rows   *sqlmock.Rows
	args   []driver.Value
	models []database.Model
}
