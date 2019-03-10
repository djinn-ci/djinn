package model

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"
)

var (
	sourceFmt = "host=%s port=%s user=%s dbname=%s password=%s sslmode=disable"

	DB *sqlx.DB

	UsersTable      = "users"
	NamespacesTable = "namespaces"
)

type Model struct {
	ID        int64      `db:"id"`
	CreatedAt *time.Time `db:"created_at"`
}

func Connect(addr, name, username, password string) error {
	host, port, err := net.SplitHostPort(addr)

	if err != nil {
		return errors.Err(err)
	}

	DB, err = sqlx.Open("postgres", fmt.Sprintf(sourceFmt, host, port, username, name, password))

	if err != nil {
		return errors.Err(err)
	}

	if err = DB.Ping(); err != nil {
		return errors.Err(err)
	}

	return nil
}

func Count(table string, cols map[string]interface{}) (int, error) {
	var count int

	query := "SELECT COUNT(*) FROM " + table + " WHERE "

	wheres := make([]string, len(cols), len(cols))
	vals := make([]interface{}, len(cols), len(cols))

	i := 0

	for key, val := range cols {
		wheres[i] = fmt.Sprintf("%s = $%d", key, i + 1)
		vals[i] = val
		i++
	}

	query += strings.Join(wheres, " AND ")

	err := DB.Get(&count, query, vals...)

	return count, errors.Err(err)
}
