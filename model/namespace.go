package model

import (
	"time"
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
