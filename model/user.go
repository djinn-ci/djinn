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
	Model

	Email     string       `db:"email"`
	Username  string       `db:"username"`
	Password  []byte       `db:"password"`
	UpdatedAt *time.Time   `db:"updated_at"`
	DeletedAt *pq.NullTime `db:"deleted_at"`
}

func FindUser(id int64) (*User, error) {
	u := &User{}

	err := DB.Get(u, "SELECT * FROM users WHERE id = $1", id)

	if err != nil {
		if err == sql.ErrNoRows {
			u.CreatedAt = nil
			u.UpdatedAt = nil
			u.DeletedAt = nil

			return u, nil
		}

		return u, errors.Err(err)
	}

	u.Email = strings.TrimSpace(u.Email)
	u.Username = strings.TrimSpace(u.Username)

	return u, nil
}

func FindUserByHandle(handle string) (*User, error) {
	u := &User{}

	err := DB.Get(u, "SELECT * FROM users WHERE email = $1 OR username = $2", handle, handle)

	if err != nil {
		if err == sql.ErrNoRows {
			u.CreatedAt = nil
			u.UpdatedAt = nil
			u.DeletedAt = nil

			return u, nil
		}

		return u, errors.Err(err)
	}

	u.Email = strings.TrimSpace(u.Email)
	u.Username = strings.TrimSpace(u.Username)

	return u, nil
}

func FindUserByUsername(username string) (*User, error) {
	u := &User{}

	err := DB.Get(u, "SELECT * FROM users WHERE username = $1", username)

	if err != nil {
		if err == sql.ErrNoRows {
			u.CreatedAt = nil
			u.UpdatedAt = nil
			u.DeletedAt = nil

			return u, nil
		}

		return u, errors.Err(err)
	}

	u.Email = strings.TrimSpace(u.Email)
	u.Username = strings.TrimSpace(u.Username)

	return u, nil
}

func (u *User) BuildsByStatus(status string) ([]*Build, error) {
	builds := make([]*Build, 0)

	err := DB.Select(&builds, `
		SELECT * FROM builds
		WHERE user_id = $1 AND status = $2
		ORDER BY created_at DESC
	`, u.ID, status)

	if err != nil {
		return builds, errors.Err(err)
	}

	for _, b := range builds {
		b.User = u
	}

	return builds, nil
}

func (u *User) BuildsByTag(tag string) ([]*Build, error) {
	builds := make([]*Build, 0)

	err := DB.Select(&builds, `
		SELECT * FROM builds
		WHERE user_id = $1 AND id in (
			SELECT build_id FROM build_tags
			WHERE name = $2
		) ORDER BY created_at ASC
	`, u.ID, tag)

	if err != nil {
		return builds, errors.Err(err)
	}

	for _, b := range builds {
		b.User = u
	}

	return builds, nil
}

func (u *User) Builds() ([]*Build, error) {
	builds := make([]*Build, 0)

	err := DB.Select(&builds, "SELECT * FROM builds WHERE user_id = $1 ORDER BY created_at DESC", u.ID)

	if err != nil {
		return builds, errors.Err(err)
	}

	for _, b := range builds {
		b.User = u
	}

	return builds, nil
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

func (u *User) FindBuild(id int64) (*Build, error) {
	b := &Build{}

	err := DB.Get(b, "SELECT * FROM builds WHERE user_id = $1 AND id = $2", u.ID, id)

	if err != nil {
		if err == sql.ErrNoRows {
			b.CreatedAt = nil
			b.StartedAt = nil
			b.FinishedAt = nil

			return b, nil
		}

		return b, errors.Err(err)
	}

	b.User = u

	return b, nil
}

func (u *User) FindNamespaceByFullName(fullName string) (*Namespace, error) {
	n := &Namespace{}

	err := DB.Get(n, "SELECT * FROM namespaces WHERE user_id = $1 AND full_name = $2", u.ID, fullName)

	if err != nil {
		if err == sql.ErrNoRows {
			n.CreatedAt = nil
			n.UpdatedAt = nil

			return n, nil
		}

		return n, errors.Err(err)
	}

	n.User = u

	return n, nil
}

func (u *User) FindOrCreateNamespace(fullName string) (*Namespace, error) {
	n, err := u.FindNamespaceByFullName(fullName)

	if err != nil {
		return n, errors.Err(err)
	}

	if !n.IsZero() {
		return n, nil
	}

	parent := &Namespace{}

	parts := strings.Split(fullName, "/")

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

func (u *User) NamespacesLike(like string) ([]*Namespace, error) {
	namespaces := make([]*Namespace, 0)

	err := DB.Select(&namespaces, `
		SELECT * FROM namespaces WHERE user_id = $1 AND full_name LIKE $2
		ORDER BY full_name ASC
	`, u.ID, "%" + like + "%")

	if err != nil {
		return namespaces, errors.Err(err)
	}

	for _, n := range namespaces {
		n.User = u
	}

	return namespaces, nil
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
