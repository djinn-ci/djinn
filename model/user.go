package model

import (
	"database/sql"
	"strconv"
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

// Get all of the builds for the current user, including the tags, namespace,
// and the namespace's user.
func (u *User) Builds() ([]*Build, error) {
	builds := make([]*Build, 0)

	buildStmt, err := DB.Preparex("SELECT * FROM builds WHERE user_id = $1 ORDER BY created_at DESC")

	if err != nil {
		return builds, errors.Err(err)
	}

	defer buildStmt.Close()

	rows, err := buildStmt.Queryx(u.ID)

	if err != nil {
		return builds, errors.Err(err)
	}

	buildIds := make([]string, 0)
	namespaceIds := make([]string, 0)

	uniq := make(map[int64]struct{})

	for rows.Next() {
		b := &Build{}

		if err := rows.StructScan(b); err != nil {
			if err == sql.ErrNoRows {
				break
			}

			return builds, errors.Err(err)
		}

		builds = append(builds, b)
		buildIds = append(buildIds, strconv.FormatInt(b.ID, 10))

		if b.NamespaceID.Valid {
			if _, ok := uniq[b.NamespaceID.Int64]; !ok {
				namespaceIds = append(namespaceIds, strconv.FormatInt(b.NamespaceID.Int64, 10))
				uniq[b.NamespaceID.Int64] = struct{}{}
			}
		}
	}

	rows, err = DB.Queryx(`
		SELECT * FROM build_tags WHERE build_id IN (` + strings.Join(buildIds, ",") + `)
		ORDER BY created_at ASC
	`)

	if err != nil {
		return builds, errors.Err(err)
	}

	tags := make([]*BuildTag, 0)

	for rows.Next() {
		t := &BuildTag{}

		if err := rows.StructScan(t); err != nil {
			if err == sql.ErrNoRows {
				break
			}

			return builds, errors.Err(err)
		}

		tags = append(tags, t)
	}

	rows, err = DB.Queryx("SELECT * FROM namespaces WHERE id IN (" + strings.Join(namespaceIds, ",") + ")")

	if err != nil {
		return builds, errors.Err(err)
	}

	namespaces := make([]*Namespace, len(namespaceIds), len(namespaceIds))
	userIds := make([]string, 0, len(namespaceIds))

	uniq = make(map[int64]struct{})

	for i := 0; rows.Next(); i++ {
		namespaces[i] = &Namespace{}

		if err := rows.StructScan(namespaces[i]); err != nil {
			if err == sql.ErrNoRows {
				break
			}

			return builds, errors.Err(err)
		}

		if _, ok := uniq[namespaces[i].UserID]; !ok {
			userIds = append(userIds, strconv.FormatInt(namespaces[i].UserID, 10))
			uniq[namespaces[i].UserID] = struct{}{}
		}
	}

	rows, err = DB.Queryx("SELECT * FROM users WHERE id IN (" + strings.Join(userIds, ",") + ")")

	if err != nil {
		return builds, errors.Err(err)
	}

	for rows.Next() {
		u := &User{}

		if err := rows.StructScan(u); err != nil {
			if err == sql.ErrNoRows {
				break
			}

			return builds, errors.Err(err)
		}

		u.Email = strings.TrimSpace(u.Email)
		u.Username = strings.TrimSpace(u.Username)

		for _, n := range namespaces {
			if n.UserID == u.ID {
				n.User = u
			}
		}
	}

	for _, b := range builds {
		for _, t := range tags {
			if b.ID == t.BuildID {
				b.Tags = append(b.Tags, t)
			}
		}

		for _, n := range namespaces {
			if n.ID == b.NamespaceID.Int64 {
				b.Namespace = n
				break
			}
		}
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
