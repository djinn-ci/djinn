package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"
)

type Tag struct {
	Model

	UserID  int64  `db:"user_id"`
	BuildID int64  `db:"build_id"`
	Name    string `db:"name"`

	User  *User
	Build *Build
}

func TagsByBuildID(id int64) ([]*Tag, error) {
	tags := make([]*Tag, 0)

	err := DB.Select(&tags, "SELECT * FROM tags WHERE build_id = $1", id)

	if err != nil {
		if err == sql.ErrNoRows {
			return tags, nil
		}

		return tags, errors.Err(err)
	}

	return tags, nil
}

func (t *Tag) Create() error {
	stmt, err := DB.Prepare(`
		INSERT INTO tags (user_id, build_id, name)
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
