package model

import (
	"database/sql"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"
)

type Namespace struct {
	Model

	UserID      int64         `db:"user_id"`
	ParentID    sql.NullInt64 `db:"parent_id"`
	Name        string        `db:"name"`
	FullName    string        `db:"full_name"`
	Description string        `db:"description"`
	Level       int64         `db:"level"`
	Visibility  Visibility    `db:"visibility"`
	UpdatedAt   *time.Time    `db:"updated_at"`

	User   *User
	Parent *Namespace
}

func FindNamespace(id int64) (*Namespace, error) {
	n := &Namespace{}

	err := DB.Get(n, "SELECT * FROM namespaces WHERE id = $1", id)

	return n, errors.Err(err)
}

func LoadNamespaceRelations(namespaces []*Namespace) error {
	if len(namespaces) == 0 {
		return nil
	}

	userIds := make([]int64, len(namespaces), len(namespaces))

	for i, n := range namespaces {
		userIds[i] = n.UserID
	}

	query, args, err := sqlx.In("SELECT * FROM users WHERE id IN (?)", userIds)

	if err != nil {
		return errors.Err(err)
	}

	users := make([]*User, 0)

	err = DB.Select(&users, DB.Rebind(query), args...)

	if err != nil {
		return errors.Err(err)
	}

	for _, n := range namespaces {
		for _, u := range users {
			if n.UserID == u.ID && n.User == nil {
				u.Email = strings.TrimSpace(u.Email)
				u.Username = strings.TrimSpace(u.Username)

				n.User = u
			}
		}
	}

	return nil
}

func (n *Namespace) Builds() ([]*Build, error) {
	builds := make([]*Build, 0)

	err := DB.Select(&builds, `
		SELECT * FROM builds WHERE namespace_id = $1 ORDER BY created_at DESC
	`, n.ID)

	if err != nil {
		return builds, errors.Err(err)
	}

	for _, b := range builds {
		b.Namespace = n
	}

	return builds, nil
}

func (n *Namespace) Create() error {
	stmt, err := DB.Prepare(`
		INSERT INTO namespaces (user_id, parent_id, name, full_name, description, level, visibility)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(
		n.UserID,
		n.ParentID,
		n.Name,
		n.FullName,
		n.Description,
		n.Level,
		n.Visibility,
	)

	err = row.Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)

	return errors.Err(err)
}

func (n *Namespace) Destroy() error {
	namespaces, err := n.Namespaces()

	if err != nil {
		return errors.Err(err)
	}

	for _, child := range namespaces {
		if err := child.Destroy(); err != nil {
			return errors.Err(err)
		}
	}

	stmt, err := DB.Prepare("DELETE FROM namespaces WHERE id = $1")

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(n.ID)

	if err != nil {
		return errors.Err(err)
	}

	n.ID = 0
	n.UserID = 0
	n.ParentID.Int64 = 0
	n.Name = ""
	n.FullName = ""
	n.Description = ""
	n.Level = 0
	n.Visibility = Visibility(0)
	n.CreatedAt = nil
	n.UpdatedAt = nil

	n.User = nil
	n.Parent = nil

	return nil
}

func (n *Namespace) FindBuild(id int64) (*Build, error) {
	b := &Build{}

	err := DB.Get(b, "SELECT * FROM builds WHERE namespace_id = $1 AND id = $2", n.ID, id)

	if err != nil {
		if err == sql.ErrNoRows {
			b.CreatedAt = nil
			b.StartedAt = nil
			b.FinishedAt = nil

			return b, nil
		}

		return b, errors.Err(err)
	}

	b.Namespace = n

	return b, nil
}

func (n Namespace) IsZero() bool {
	return	n.ID == 0                     &&
			n.UserID == 0                 &&
			n.Name == ""                  &&
			n.FullName == ""              &&
			n.Description == ""           &&
			n.Level == 0                  &&
			n.Visibility == Visibility(0) &&
			n.CreatedAt == nil            &&
			n.UpdatedAt == nil
}

func (n *Namespace) LoadParents() error {
	if n.ParentID.Int64 == 0 {
		return nil
	}

	parent := &Namespace{}

	err := DB.Get(parent, "SELECT * FROM namespaces WHERE id = $1", n.ParentID)

	if err != nil {
		return errors.Err(err)
	}

	if parent.IsZero() {
		return nil
	}

	n.Parent = parent

	return n.Parent.LoadParents()
}

func (n Namespace) NamespacesLike(like string) ([]*Namespace, error) {
	namespaces := make([]*Namespace, 0)

	err := DB.Select(&namespaces, `
		SELECT * FROM namespaces WHERE parent_id = $1  AND full_name LIKE $2
	`, n.ID, "%" + like + "%")

	if err != nil {
		return namespaces, errors.Err(err)
	}

	if err := LoadNamespaceRelations(namespaces); err != nil {
		return namespaces, errors.Err(err)
	}

	return namespaces, nil
}

func (n Namespace) Namespaces() ([]*Namespace, error) {
	namespaces := make([]*Namespace, 0)

	err := DB.Select(&namespaces, `
		SELECT * FROM namespaces WHERE parent_id = $1 ORDER BY full_name ASC
	`, n.ID)

	if err != nil {
		return namespaces, errors.Err(err)
	}

	if err := LoadNamespaceRelations(namespaces); err != nil {
		return namespaces, errors.Err(err)
	}

	return namespaces, nil
}

func (n *Namespace) Update() error {
	stmt, err := DB.Prepare(`
		UPDATE namespaces
		SET	name = $1,
			full_name = $2,
			description = $3,
			visibility = $4,
			updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(n.Name, n.FullName, n.Description, n.Visibility, n.ID)

	err = row.Scan(&n.UpdatedAt)

	return errors.Err(err)
}

func (n Namespace) URI() string {
	if n.User == nil {
		return ""
	}

	return "/u/" + n.User.Username + "/" + n.FullName
}
