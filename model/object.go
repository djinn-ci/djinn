package model

import (
	"github.com/andrewpillar/thrall/errors"

	"github.com/lib/pq"
)

type Object struct {
	Model

	UserID    int64        `db:"user_id"`
	Name      string       `db:"name"`
	Filename  string       `db:"filename"`
	Type      string       `db:"type"`
	Size      int64        `db:"size"`
	MD5       []byte       `db:"md5"`
	SHA256    []byte       `db:"sha256"`
	DeletedAt *pq.NullTime `db:"deleted_at"`

	User *User
}

func (o *Object) Create() error {
	stmt, err := DB.Prepare(`
		INSERT INTO objects (user_id, name, filename, type, size)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	err = stmt.QueryRow(o.UserID, o.Name, o.Filename, o.Type, o.Size).Scan(&o.ID, &o.CreatedAt)

	return errors.Err(err)
}

func (o *Object) IsZero() bool {
	return	o.ID == 0                                &&
			o.UserID == 0                            &&
			o.Name == ""                             &&
			o.Filename == ""                         &&
			o.Type == ""                             &&
			o.Size == 0                              &&
			len(o.MD5) == 0                          &&
			len(o.SHA256) == 0                       &&
			o.DeletedAt == nil || !o.DeletedAt.Valid
}
