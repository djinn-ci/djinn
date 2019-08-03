package model

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"
)

type Tag struct {
	Model

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

func (t *Tag) Destroy() error {
	q := Delete(
		Table("tags"),
		WhereEq("id", t.ID),
	)

	stmt, err := t.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(q.Args()...)

	return errors.Err(err)
}

func (t Tag) UIEndpoint(uri ...string) string {
	endpoint := fmt.Sprintf("/builds/%v/tags/%v", t.BuildID, t.ID)

	if len(uri) > 0 {
		endpoint = fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}

	return endpoint
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

func (ts TagStore) Find(id int64) (*Tag, error) {
	t := &Tag{
		Model: Model{
			DB: ts.DB,
		},
		Build: ts.Build,
		User:  ts.User,
	}

	q := Select(
		Columns("*"),
		Table("tags"),
		WhereEq("id", id),
		ForBuild(ts.Build),
	)

	err := ts.Get(t, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return t, errors.Err(err)
}

func (ts TagStore) Index(opts ...Option) ([]*Tag, error) {
	tt, err := ts.All(opts...)

	if err != nil {
		return tt, errors.Err(err)
	}

	err = ts.LoadUsers(tt)

	return tt, errors.Err(err)
}

func (ts TagStore) LoadUsers(tt []*Tag) error {
	if len(tt) == 0 {
		return nil
	}

	ids := make([]interface{}, len(tt), len(tt))

	for i, t := range tt {
		ids[i] = t.UserID
	}

	users := UserStore{
		DB: ts.DB,
	}

	uu, err := users.All(WhereIn("id", ids...))

	if err != nil {
		return errors.Err(err)
	}

	for _, t := range tt {
		for _, u := range uu {
			if t.UserID == u.ID {
				t.User = u
			}
		}
	}

	return nil
}

func (ts TagStore) New() *Tag {
	t := &Tag{
		Model: Model{
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
