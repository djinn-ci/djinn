package provider

import (
	"database/sql/driver"

	"github.com/andrewpillar/thrall/database"

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

var (
	insertFmt = "INSERT INTO %s \\([\\w+, ]+\\) VALUES \\([\\$\\d+, ]+\\) RETURNING id"
	updateFmt = "UPDATE %s SET [\\w+ \\= \\$\\d+,]+ WHERE \\([\\w]+ \\= \\$\\d+\\)"
	deleteFmt = "DELETE FROM %s WHERE \\(id IN \\([\\$\\d+ , ]+\\)\\)"
)
