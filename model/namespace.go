package model

import (
	"database/sql"
	"time"

	"github.com/andrewpillar/thrall/errors"
)

type Namespace struct {
	ID          int64         `db:"id"`
	UserID      int64         `db:"user_id"`
	ParentID    sql.NullInt64 `db:"parent_id"`
	Name        string        `db:"name"`
	FullName    string        `db:"full_name"`
	Description string        `db:"description"`
	Level       int64         `db:"level"`
	Visibility  Visibility    `db:"visibility"`
	CreatedAt   *time.Time    `db:"created_at"`
	UpdatedAt   *time.Time    `db:"updated_at"`

	User   *User
	Parent *Namespace
}

func FindNamespace(id int64) (*Namespace, error) {
	n := &Namespace{}

	stmt, err := DB.Prepare(`SELECT * FROM namespaces WHERE id = $1`)

	if err != nil {
		return n, errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(id)

	err = row.Scan(&n.ID, &n.UserID, &n.ParentID, &n.Name, &n.FullName, &n.Description, &n.Level, &n.Visibility, &n.CreatedAt, &n.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return n, nil
		}

		return n, errors.Err(err)
	}

	return n, nil
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

	row := stmt.QueryRow(n.UserID, n.ParentID, n.Name, n.FullName, n.Description, n.Level, n.Visibility)

	err = row.Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)

	return errors.Err(err)
}

func (n *Namespace) Destroy() error {
	stmt, err := DB.Prepare(`DELETE FROM namespaces WHERE id = $1`)

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

	stmt, err := DB.Prepare(`SELECT * FROM namespaces WHERE id = $1`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(n.ParentID)

	p := &Namespace{}

	err = row.Scan(&p.ID, &p.UserID, &p.ParentID, &p.Name, &p.FullName, &p.Description, &p.Level, &p.Visibility, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		return errors.Err(err)
	}

	n.Parent = p

	return n.Parent.LoadParents()
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
