package model

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type Build struct {
	ID          int64         `db:"id"`
	UserID      int64         `db:"user_id"`
	NamespaceID sql.NullInt64 `db:"namespace_id"`
	Manifest    string        `db:"manifest"`
	Status      Status        `db:"status"`
	Output      string        `db:"output"`
	CreatedAt   *time.Time    `db:"created_at"`
	StartedAt   *pq.NullTime  `db:"started_at"`
	FinishedAt  *pq.NullTime  `db:"finished_at"`
}
