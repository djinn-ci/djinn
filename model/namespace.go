package model

import (
	"time"

	"github.com/andrewpillar/thrall/errors"
)

type Namespace struct {
	ID          int64      `db:"id"`
	UserID      int64      `db:"user_id"`
	Name        string     `db:"name"`
	Description string     `db:"description"`
	Private     bool       `db:"private"`
	CreatedAt   *time.Time `db:"created_at"`
	UpdatedAt   *time.Time `db:"updated_at"`
}

func (n *Namespace) Create() error {
	stmt, err := DB.Prepare(`
		INSERT INTO namespaces (user_id, name, description, private)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(n.UserID, n.Name, n.Description, n.Private)

	err = row.Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)

	return errors.Err(err)
}
