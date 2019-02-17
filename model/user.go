package model

import (
	"database/sql"
	"time"

	"github.com/andrewpillar/thrall/errors"

	"github.com/lib/pq"
)

type User struct {
	ID        int64        `db:"id"`
	Email     string       `db:"email"`
	Username  string       `db:"username"`
	Password  []byte       `db:"password"`
	CreatedAt *time.Time   `db:"created_at"`
	UpdatedAt *time.Time   `db:"updated_at"`
	DeletedAt *pq.NullTime `db:"deleted_at"`
}

// FindUserByHandle is only called during authentication. Therefore we only
// want to populate the ID field for setting the session, and the password
// field for performing the actual authentication.
func FindUserByHandle(handle string) (*User, error) {
	u := &User{}

	stmt, err := DB.Prepare(`SELECT id, password FROM users WHERE email = $1 OR username = $2`)

	if err != nil {
		return u, errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(handle, handle)

	err = row.Scan(&u.ID, &u.Password)

	if err == sql.ErrNoRows {
		return u, nil
	}

	return u, errors.Err(err)
}

func (u *User) Create() error {
	stmt, err := DB.Prepare(`
		INSERT INTO users (email, username, password)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at, deleted_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(u.Email, u.Username, u.Password)

	err = row.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)

	return errors.Err(err)
}
