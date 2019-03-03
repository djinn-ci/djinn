package model

import (
	"database/sql"
	"time"

	"github.com/andrewpillar/thrall/errors"

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

type BuildTag struct {
	ID      int64        `db:"id"`
	UserID  int64        `db:"user_id"`
	BuildID int64        `db:"build_id"`
	Name    string       `db:"name"`
	CreatedAt *time.Time `db:"created_at"`
}

func (b *Build) Create() error {
	stmt, err := DB.Prepare(`
		INSERT INTO builds (user_id, namespace_id, manifest)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	err = stmt.QueryRow(b.UserID, b.NamespaceID, b.Manifest).Scan(&b.ID, &b.CreatedAt)

	return errors.Err(err)
}

func (t *BuildTag) Create() error {
	stmt, err := DB.Prepare(`
		INSERT INTO build_tags (user_id, build_id, name)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	err = stmt.QueryRow(t.UserID, t.BuildID, t.Name).Scan(&t.ID, &t.CreatedAt)

	return errors.Err(err)
}
