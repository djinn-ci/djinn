package variable

import (
	"database/sql/driver"

	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"
)

type testQuery struct {
	query  string
	opts   []query.Option
	rows   *sqlmock.Rows
	args   []driver.Value
	models []model.Model
}

var (
	insertFmt = "INSERT INTO %s \\([\\w+, ]+\\) VALUES \\([\\$\\d+, ]+\\) RETURNING id"
	deleteFmt = "DELETE FROM %s WHERE \\(id IN \\([\\$\\d+ , ]+\\)\\)"
)
