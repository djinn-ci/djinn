package namespace

import (
	"database/sql/driver"

	"djinn-ci.com/database"

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
	updateFmt = "UPDATE %s SET [a-z_ \\= \\$\\d+,]+ WHERE \\([a-z_]+ \\= \\$\\d\\)"
	deleteFmt = "DELETE FROM %s WHERE \\(id IN \\([\\$\\d+ , ]+\\)\\)"
)
