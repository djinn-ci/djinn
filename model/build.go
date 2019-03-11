package model

import (
	"bytes"
	"database/sql"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/errors"

	"github.com/lib/pq"
)

type Build struct {
	Model

	UserID      int64          `db:"user_id"`
	NamespaceID sql.NullInt64  `db:"namespace_id"`
	Manifest    string         `db:"manifest"`
	Status      Status         `db:"status"`
	Output      sql.NullString `db:"output"`
	StartedAt   *pq.NullTime   `db:"started_at"`
	FinishedAt  *pq.NullTime   `db:"finished_at"`

	User      *User
	Namespace *Namespace
	Tags      []*BuildTag
}

type BuildTag struct {
	Model

	UserID  int64  `db:"user_id"`
	BuildID int64  `db:"build_id"`
	Name    string `db:"name"`
}

func LoadBuildRelations(builds []*Build) error {
	if len(builds) == 0 {
		return nil
	}

	namespacesQuery := bytes.NewBufferString("SELECT * FROM namespaces WHERE id IN (")
	usersQuery := bytes.NewBufferString("SELECT * FROM users WHERE id IN (")
	tagsQuery := bytes.NewBufferString("SELECT * FROM build_tags WHERE build_id IN (")

	namespaceIds := make([]string, 0)

	end := len(builds) - 1
	loadNamespaces := false

	for i, b := range builds {
		if b.NamespaceID.Valid {
			namespaceIds = append(namespaceIds, strconv.FormatInt(b.NamespaceID.Int64, 10))
			loadNamespaces = true
		}

		usersQuery.WriteString(strconv.FormatInt(b.UserID, 10))
		tagsQuery.WriteString(strconv.FormatInt(b.ID, 10))

		if i != end {
			usersQuery.WriteString(", ")
			tagsQuery.WriteString(", ")
		}
	}

	namespacesQuery.WriteString(strings.Join(namespaceIds, ", ") + ")")
	usersQuery.WriteString(")")
	tagsQuery.WriteString(")")

	namespaces := make([]*Namespace, 0)
	users := make([]*User, 0)
	tags := make([]*BuildTag, 0)

	if loadNamespaces {
		if err := DB.Select(&namespaces, namespacesQuery.String()); err != nil {
			return errors.Err(err)
		}
	}

	if err := DB.Select(&users, usersQuery.String()); err != nil {
		return errors.Err(err)
	}

	if err := DB.Select(&tags, tagsQuery.String()); err != nil {
		return errors.Err(err)
	}

	if err := LoadNamespaceRelations(namespaces); err != nil {
		return errors.Err(err)
	}

	for _, b := range builds {
		for _, t := range tags {
			if t.BuildID == b.ID {
				b.Tags = append(b.Tags, t)
			}
		}

		for _, n := range namespaces {
			if b.NamespaceID.Valid && b.NamespaceID.Int64 == n.ID && b.Namespace == nil {
				b.Namespace = n
			}
		}

		for _, u := range users {
			if b.UserID == u.ID && b.User == nil {
				b.User = u
			}
		}
	}

	return nil
}

func (b *Build) Create() error {
	stmt, err := DB.Prepare(`
		INSERT INTO builds (user_id, namespace_id, manifest)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	err = stmt.QueryRow(b.UserID, b.NamespaceID, b.Manifest).Scan(&b.ID, &b.CreatedAt)

	return errors.Err(err)
}

func (b *Build) IsZero() bool {
	return	b.ID == 0                &&
			b.UserID == 0            &&
			b.NamespaceID.Int64 == 0 &&
			!b.NamespaceID.Valid     &&
			b.Manifest == ""         &&
			b.Status == Status(0)    &&
			b.Output.String == ""    &&
			!b.Output.Valid          &&
			b.CreatedAt == nil       &&
			b.StartedAt == nil
}

func (b *Build) URI() string {
	return "/builds/" + strconv.FormatInt(b.ID, 10)
}

func (t *BuildTag) Create() error {
	stmt, err := DB.Prepare(`
		INSERT INTO build_tags (user_id, build_id, name)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	err = stmt.QueryRow(t.UserID, t.BuildID, t.Name).Scan(&t.ID, &t.CreatedAt)

	return errors.Err(err)
}
