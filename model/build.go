package model

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"

	"github.com/lib/pq"
)

type Build struct {
	ID          int64          `db:"id"`
	UserID      int64          `db:"user_id"`
	NamespaceID sql.NullInt64  `db:"namespace_id"`
	Manifest    string         `db:"manifest"`
	Status      Status         `db:"status"`
	Output      sql.NullString `db:"output"`
	CreatedAt   *time.Time     `db:"created_at"`
	StartedAt   *pq.NullTime   `db:"started_at"`
	FinishedAt  *pq.NullTime   `db:"finished_at"`

	User      *User
	Namespace *Namespace
	Tags      []*BuildTag
}

type BuildTag struct {
	ID      int64        `db:"id"`
	UserID  int64        `db:"user_id"`
	BuildID int64        `db:"build_id"`
	Name    string       `db:"name"`
	CreatedAt *time.Time `db:"created_at"`
}

func BuildsWithRelations(col string, val interface{}) ([]*Build, error) {
	builds := make([]*Build, 0)

	buildStmt, err := DB.Preparex("SELECT * FROM builds WHERE " + col + " = $1 ORDER BY created_at DESC")

	if err != nil {
		return builds, errors.Err(err)
	}

	defer buildStmt.Close()

	rows, err := buildStmt.Queryx(val)

	if err != nil {
		return builds, errors.Err(err)
	}

	buildIds := make([]string, 0)
	namespaceIds := make([]string, 0)

	uniq := make(map[int64]struct{})

	for rows.Next() {
		b := &Build{}

		if err := rows.StructScan(b); err != nil {
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

	if len(builds) == 0 {
		return builds, nil
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
