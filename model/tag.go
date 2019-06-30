package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"
)

type Tag struct {
	model

	UserID  int64  `db:"user_id"`
	BuildID int64  `db:"build_id"`
	Name    string `db:"name"`

	User  *User
	Build *Build
}

type TagStore struct {
	*sqlx.DB

	User  *User
	Build *Build
}

func (t *Tag) Create() error {
	q := Insert(
		Table("tags"),
		Columns("user_id", "build_id", "name"),
		Values(t.UserID, t.BuildID, t.Name),
		Returning("id", "created_at", "updated_at"),
	)

	stmt, err := t.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt))
}

func (ts TagStore) All(opts ...Option) ([]*Tag, error) {
	tt := make([]*Tag, 0)

	opts = append([]Option{Columns("*")}, opts...)

	q := Select(append(opts, ForBuild(ts.Build), Table("tags"))...)

	err := ts.Select(&tt, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, t := range tt {
		t.DB = ts.DB

		if ts.Build != nil {
			t.Build = ts.Build
		}
	}

	return tt, errors.Err(err)
}

func (ts TagStore) New() *Tag {
	t := &Tag{
		model: model{
			DB: ts.DB,
		},
		User:  ts.User,
		Build: ts.Build,
	}

	if ts.Build != nil {
		t.BuildID = ts.Build.ID
	}

	if ts.User != nil {
		t.UserID = ts.User.ID
	}

	return t
}
