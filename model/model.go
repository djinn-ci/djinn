package model

import (
	"fmt"
	"net"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"
)

var (
	sourceFmt = "host=%s port=%s user=%s dbname=%s password=%s sslmode=disable"

	DB *sqlx.DB

	UsersTable = "users"
)

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

func Count(table, col, val string) (int, error) {
	var count int

	if err := DB.Get(&count, "SELECT COUNT(*) FROM " + table + " WHERE " + col + " = $1", val); err != nil {
		return count, errors.Err(err)
	}

	return count, nil
}
