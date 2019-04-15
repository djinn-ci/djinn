package model

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"

	"github.com/RichardKnop/machinery/v1/tasks"
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
	Tags      []*Tag
	Stages    []*Stage
}

func LoadBuildRelations(builds []*Build) error {
	if len(builds) == 0 {
		return nil
	}

	namespaceIds := make([]int64, 0, len(builds))
	buildIds := make([]int64, len(builds), len(builds))
	userIds := make([]int64, len(builds), len(builds))

	for i, b := range builds {
		if b.NamespaceID.Valid {
			namespaceIds = append(namespaceIds, b.NamespaceID.Int64)
		}

		buildIds[i] = b.ID
		userIds[i] = b.UserID
	}

	namespaces := make([]*Namespace, 0)

	if len(namespaceIds) > 0 {
		query, args, err := sqlx.In("SELECT * FROM namespaces WHERE id IN (?)", namespaceIds)

		if err != nil {
			return errors.Err(err)
		}

		err = DB.Select(&namespaces, DB.Rebind(query), args...)

		if err != nil {
			return errors.Err(err)
		}
	}

	query, args, err := sqlx.In("SELECT * FROM tags WHERE build_id IN (?)", buildIds)

	if err != nil {
		return errors.Err(err)
	}

	tags := make([]*Tag, 0)

	err = DB.Select(&tags, DB.Rebind(query), args...)

	if err != nil {
		return errors.Err(err)
	}

	query, args, err = sqlx.In("SELECT * FROM users WHERE id IN (?)", userIds)

	if err != nil {
		return errors.Err(err)
	}

	users := make([]*User, 0)

	err = DB.Select(&users, DB.Rebind(query), args...)

	if err != nil {
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

func (b *Build) FindJob(id int64) (*Job, error) {
	j := &Job{}

	err := DB.Get(j, "SELECT * FROM jobs WHERE build_id = $1 AND id = $2", b.ID, id)

	if err != nil {
		if err == sql.ErrNoRows {
			return j, nil
		}

		return j, errors.Err(err)
	}

	j.Build = b

	return j, nil
}

func (b *Build) IsZero() bool {
	return	b.ID == 0                                         &&
			b.UserID == 0                                     &&
			b.NamespaceID.Int64 == 0                          &&
			!b.NamespaceID.Valid                              &&
			b.Manifest == ""                                  &&
			b.Status == Status(0)                             &&
			b.Output.String == ""                             &&
			!b.Output.Valid                                   &&
			b.CreatedAt == nil || *b.CreatedAt == time.Time{} &&
			b.StartedAt.Time == time.Time{}                   &&
			!b.StartedAt.Valid                                &&
			b.FinishedAt.Time == time.Time{}                  &&
			!b.FinishedAt.Valid
}

func (b *Build) LoadRelations() error {
	if b.User == nil {
		b.User = &User{}

		err := DB.Get(b.User, "SELECT * FROM users WHERE id = $1", b.UserID)

		if err != nil {
			return errors.Err(err)
		}
	}

	if b.NamespaceID.Valid {
		b.Namespace = &Namespace{}

		err := DB.Get(b.Namespace, "SELECT * FROM namespaces WHERE id = $1", b.NamespaceID.Int64)

		if err != nil {
			return errors.Err(err)
		}
	}

	b.Tags = make([]*Tag, 0)

	err := DB.Select(&b.Tags, "SELECT * FROM tags WHERE build_id = $1", b.ID)

	if err != nil {
		return errors.Err(err)
	}

	b.Stages = make([]*Stage, 0)

	err = DB.Select(&b.Stages, "SELECT * FROM stages WHERE build_id = $1", b.ID)

	return errors.Err(err)
}

func (b Build) Signature() *tasks.Signature {
	id := tasks.Arg{
		Type:  "int64",
		Value: b.ID,
	}

	manifest := tasks.Arg{
		Type:  "string",
		Value: b.Manifest,
	}

	return &tasks.Signature{
		Name: "build",
		Args: []tasks.Arg{id, manifest},
	}
}

func (b *Build) URI() string {
	return "/builds/" + strconv.FormatInt(b.ID, 10)
}
