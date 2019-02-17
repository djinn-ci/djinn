package model

import (
	"time"
)

type Namespace struct {
	ID         int64     `db:"id"`
	UserID     int64     `db:"user_id"`
	Name       string    `db:"name"`
	CreatedAt *time.Time `db:"created_at"`
}
