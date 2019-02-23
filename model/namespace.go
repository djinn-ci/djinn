package model

import (
	"database/sql/driver"
	"time"

	"github.com/andrewpillar/thrall/errors"
)

type Visibility uint8

const (
	Private Visibility	= iota
	Internal
	Public
)

type Namespace struct {
	ID          int64      `db:"id"`
	UserID      int64      `db:"user_id"`
	Name        string     `db:"name"`
	Description string     `db:"description"`
	Visibility  Visibility `db:"visibility"`
	CreatedAt   *time.Time `db:"created_at"`
	UpdatedAt   *time.Time `db:"updated_at"`

	User *User
}

// Parse the visibility from the given string. If the level of visibility could
// not be determined then it will default to Private.
func ParseVisibility(s string) Visibility {
	switch s {
		case "private":
			return Private
		case "internal":
			return Internal
		case "public":
			return Public
		default:
			return Private
	}
}

func (v Visibility) Value() (driver.Value, error) {
	switch v {
		case Private:
			return driver.Value("private"), nil
		case Internal:
			return driver.Value("internal"), nil
		case Public:
			return driver.Value("public"), nil
		default:
			return driver.Value(""), errors.Err(errors.New("unknown visibility level"))
	}
}

func (n *Namespace) Create() error {
	stmt, err := DB.Prepare(`
		INSERT INTO namespaces (user_id, name, description, visibility)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(n.UserID, n.Name, n.Description, n.Visibility)

	err = row.Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)

	return errors.Err(err)
}

func (n Namespace) IsZero() bool {
	return	n.ID == 0                     &&
			n.UserID == 0                 &&
			n.Name == ""                  &&
			n.Description == ""           &&
			n.Visibility == Visibility(0) &&
			n.CreatedAt == nil            &&
			n.UpdatedAt == nil
}

func (v *Visibility) Scan(val interface{}) error {
	if val == nil {
		(*v) = Visibility(0)
		return nil
	}

	sval, err := driver.String.ConvertValue(val)

	if err != nil {
		return errors.Err(err)
	}

	s, ok := sval.([]byte)

	if !ok {
		return errors.Err(errors.New("failed to scan Visibility"))
	}

	(*v) = ParseVisibility(string(s))

	return nil
}
