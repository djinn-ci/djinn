package model

import (
	"database/sql"
	"regexp"
	"strings"
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

func FindUser(id int64) (*User, error) {
	u := &User{}

	stmt, err := DB.Prepare(`SELECT * FROM users WHERE id = $1`)

	if err != nil {
		return u, errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(id)

	err = row.Scan(&u.ID, &u.Email, &u.Username, &u.Password, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return u, nil
		}

		return u, errors.Err(err)
	}

	u.Email = strings.TrimSpace(u.Email)
	u.Username = strings.TrimSpace(u.Username)

	return u, nil
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

	if err != nil {
		if err == sql.ErrNoRows {
			return u, nil
		}
	}

	return u, errors.Err(err)
}

func FindUserByUsername(username string) (*User, error) {
	u := &User{}

	stmt, err := DB.Prepare(`SELECT * FROM users WHERE username = $1`)

	if err != nil {
		return u, errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(username)

	err = row.Scan(&u.ID, &u.Email, &u.Username, &u.Password, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return u, nil
		}
	}

	u.Email = strings.TrimSpace(u.Email)
	u.Username = strings.TrimSpace(u.Username)

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

func (u *User) FindNamespaceByFullName(fullName string) (*Namespace, error) {
	n := &Namespace{}

	stmt, err := DB.Prepare(`SELECT * FROM namespaces WHERE user_id = $1 AND full_name = $2`)

	if err != nil {
		return n, errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(u.ID, fullName)

	err = row.Scan(&n.ID, &n.UserID, &n.ParentID, &n.Name, &n.FullName, &n.Description, &n.Level, &n.Visibility, &n.CreatedAt, &n.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return n, nil
		}
	}

	n.User = u

	return n, errors.Err(err)
}

func (u *User) FindOrCreateNamespace(fullName string) (*Namespace, error) {
	n, err := u.FindNamespaceByFullName(fullName)

	if err != nil {
		return n, errors.Err(err)
	}

	if !n.IsZero() {
		return n, nil
	}

	parts := strings.Split(fullName, "/")

	parent := &Namespace{}

	for _, name := range parts {
		if parent.Level + 1 > 20 {
			break
		}

		if matched, err := regexp.Match("^[a-zA-Z0-9]+$", []byte(name)); !matched || err != nil {
			break
		}

		n = &Namespace{
			UserID:   u.ID,
			Name:     name,
			FullName: name,
			Level:    parent.Level + 1,
		}

		if !parent.IsZero() {
			n.ParentID = sql.NullInt64{
				Int64: parent.ID,
				Valid: true,
			}

			n.FullName = strings.Join([]string{parent.FullName, n.Name}, "/")
		}

		if err := n.Create(); err != nil {
			return n, errors.Err(err)
		}

		parent = n
	}

	return n, nil
}

func (u User) IsZero() bool {
	return	u.ID == 0            &&
			u.Email == ""        &&
			u.Username == ""     &&
			len(u.Password) == 0 &&
			u.CreatedAt == nil   &&
			u.UpdatedAt == nil
}

func (u *User) Namespaces() ([]*Namespace, error) {
	namespaces := make([]*Namespace, 0)

	err := DB.Select(&namespaces, "SELECT * FROM namespaces WHERE user_id = $1 ORDER BY full_name ASC", u.ID)

	if err != nil {
		return namespaces, errors.Err(err)
	}

	for _, n := range namespaces {
		n.User = u
	}

	return namespaces, nil
}
