package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"
)

type Artifact struct {
	Model

	BuildID int64          `db:"build_id"`
	JobID   int64          `db:"job_id"`
	Source  string         `db:"source"`
	Name    string         `db:"name"`
	Size    sql.NullInt64  `db:"size"`
	Type    sql.NullString `db:"type"`
	MD5     []byte         `db:"md5"`
	SHA256  []byte         `db:"sha256"`
}

func (a *Artifact) Create() error {
	stmt, err := DB.Prepare(`
		INSERT INTO artifacts (build_id, job_id, source, name, size, type, md5, sha256)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(a.BuildID, a.JobID, a.Source, a.Name, a.Size, a.Type, a.MD5, a.SHA256)

	err = row.Scan(&a.ID, &a.CreatedAt)

	return errors.Err(err)
}
