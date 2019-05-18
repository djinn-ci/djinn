package model

import (
	"fmt"
	"database/sql"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type Build struct {
	model

	UserID      int64          `db:"user_id"`
	NamespaceID sql.NullInt64  `db:"namespace_id"`
	Manifest    string         `db:"manifest"`
	Status      runner.Status  `db:"status"`
	Output      sql.NullString `db:"output"`
	StartedAt   *pq.NullTime   `db:"started_at"`
	FinishedAt  *pq.NullTime   `db:"finished_at"`

	User      *User
	Namespace *Namespace
	Driver    *Driver
	Tags      []*Tag
	Stages    []*Stage
	Objects   []*BuildObject
	Artifacts []*Artifact
	Variables []*BuildVariable
}

type BuildStore struct {
	*sqlx.DB

	user      *User
	namespace *Namespace
}

func (bs BuildStore) New() *Build {
	b := &Build{
		model: model{
			DB: bs.DB,
		},
		User:      bs.user,
		Namespace: bs.namespace,
	}

	if bs.user != nil {
		b.UserID = bs.user.ID
	}

	if bs.namespace != nil {
		b.NamespaceID = sql.NullInt64{
			Int64: bs.namespace.ID,
			Valid: true,
		}
	}

	return b
}

func (bs BuildStore) All() ([]*Build, error) {
	bb := make([]*Build, 0)

	query := "SELECT * FROM builds"
	args := []interface{}{}

	if bs.user != nil {
		query += " WHERE user_id = $1"
		args = append(args, bs.user.ID)
	}

	if bs.namespace != nil {
		if bs.user != nil {
			query += " AND namespace_id = $2"
		} else {
			query += " WHERE namespace_id = $1"
		}

		args = append(args, bs.namespace.ID)
	}

	err := bs.Select(&bb, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, b := range bb {
		b.DB = bs.DB

		if bs.user != nil {
			b.User = bs.user
		}

		if bs.namespace != nil {
			b.Namespace = bs.namespace
		}
	}

	return bb, errors.Err(err)
}

func (bs BuildStore) ByStatus(status string) ([]*Build, error) {
	bb := make([]*Build, 0)

	query := "SELECT * FROM builds WHERE status = $1"
	args := []interface{}{status}

	if bs.user != nil {
		query += " AND user_id = $2"
		args = append(args, bs.user.ID)
	}

	if bs.namespace != nil {
		if bs.user != nil {
			query += " AND namespace_id = $3"
		} else {
			query += " AND namespace_id = $2"
		}

		args = append(args, bs.namespace.ID)
	}

	err := bs.Select(&bb, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, b := range bb {
		b.DB = bs.DB

		if bs.user != nil {
			b.User = bs.user
		}

		if bs.namespace != nil {
			b.Namespace = bs.namespace
		}
	}

	return bb, errors.Err(err)
}

func (bs BuildStore) Find(id int64) (*Build, error) {
	b := &Build{
		model: model{
			DB: bs.DB,
		},
	}

	query := "SELECT * FROM builds WHERE id = $1"
	args := []interface{}{id}

	if bs.user != nil {
		query += " AND user_id = $2"
		args = append(args, bs.user.ID)

		b.User = bs.user
	}

	if bs.namespace != nil {
		if bs.user != nil {
			query += " AND namespace_id = $3"
		} else {
			query += " AND namespace_id = $2"
		}

		args = append(args, bs.namespace.ID)

		b.Namespace = bs.namespace
	}

	err := bs.Get(b, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return b, errors.Err(err)
}

func (bs *BuildStore) In(ids ...int64) ([]*Build, error) {
	bb := make([]*Build, 0)

	if len(ids) == 0 {
		return bb, nil
	}

	query, args, err := sqlx.In("SELECT * FROM builds WHERE id IN (?)", ids)

	if err != nil {
		return bb, errors.Err(err)
	}

	err = bs.Select(&bb, bs.Rebind(query), args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, b := range bb {
		b.DB = bs.DB
	}

	return bb, errors.Err(err)
}

func (bs *BuildStore) LoadNamespaces(bb []*Build) error {
	if len(bb) == 0 {
		return nil
	}

	ids := make([]int64, len(bb), len(bb))

	for i, b := range bb {
		if b.NamespaceID.Valid {
			ids[i] = b.NamespaceID.Int64
		}
	}

	namespaces := NamespaceStore{
		DB: bs.DB,
	}

	nn, err := namespaces.In(ids...)

	if err != nil {
		return errors.Err(err)
	}

	for _, b := range bb {
		for _, n := range nn {
			if b.NamespaceID.Int64 == n.ID {
				b.Namespace = n
			}
		}
	}

	return nil
}

func (bs *BuildStore) LoadTags(bb []*Build) error {
	if len(bb) == 0 {
		return nil
	}

	ids := make([]int64, len(bb), len(bb))

	for i, b := range bb {
		ids[i] = b.ID
	}

	tags := TagStore{
		DB: bs.DB,
	}

	tt, err := tags.InBuildID(ids...)

	if err != nil {
		return errors.Err(err)
	}

	for _, b := range bb {
		for _, t := range tt {
			if b.ID == t.BuildID {
				b.Tags = append(b.Tags, t)
			}
		}
	}

	return nil
}

func (bs *BuildStore) LoadUsers(bb []*Build) error {
	if len(bb) == 0 {
		return nil
	}

	ids := make([]int64, len(bb), len(bb))

	for i, b := range bb {
		ids[i] = b.UserID
	}

	users := UserStore{
		DB: bs.DB,
	}

	uu, err := users.In(ids...)

	if err != nil {
		return errors.Err(err)
	}

	for _, b := range bb {
		for _, u := range uu {
			if b.UserID == u.ID {
				b.User = u
			}
		}
	}

	return nil
}

func (b *Build) ArtifactStore() ArtifactStore {
	return ArtifactStore{
		DB:    b.DB,
		build: b,
	}
}

func (b *Build) DriverStore() DriverStore {
	return DriverStore{
		DB:    b.DB,
		build: b,
	}
}

func (b *Build) TagStore() TagStore {
	return TagStore{
		DB:    b.DB,
		user:  b.User,
		build: b,
	}
}

func (b *Build) StageStore() StageStore {
	return StageStore{
		DB:    b.DB,
		build: b,
	}
}

func (b *Build) JobStore() JobStore {
	return JobStore{
		DB:    b.DB,
		build: b,
	}
}

func (b *Build) BuildObjectStore() BuildObjectStore {
	return BuildObjectStore{
		DB:    b.DB,
		build: b,
	}
}

func (b *Build) BuildVariableStore() BuildVariableStore {
	return BuildVariableStore{
		DB:    b.DB,
		build: b,
	}
}

func (b *Build) Create() error {
	stmt, err := b.Prepare(`
		INSERT INTO builds (user_id, namespace_id, manifest)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(b.UserID, b.NamespaceID, b.Manifest)

	return errors.Err(row.Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt))
}

func (b *Build) Update() error {
	stmt, err := b.Prepare(`
		UPDATE builds
		SET status = $1, output = $2, started_at = $3, finished_at = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(b.Status, b.Output, b.StartedAt, b.FinishedAt, b.ID)

	return errors.Err(row.Scan(&b.UpdatedAt))
}

func (b *Build) IsZero() bool {
	return b.ID == 0 &&
           b.UserID == 0 &&
           !b.NamespaceID.Valid &&
           b.Manifest == "" &&
           b.Status == runner.Status(0) &&
           !b.Output.Valid &&
           b.StartedAt == nil &&
           b.FinishedAt == nil &&
           b.CreatedAt == nil &&
           b.UpdatedAt == nil
}

func (b *Build) LoadUser() error {
	var err error

	users := UserStore{
		DB: b.DB,
	}

	b.User, err = users.Find(b.UserID)

	return errors.Err(err)
}

func (b *Build) LoadNamespace() error {
	var err error

	namespaces := NamespaceStore{
		DB: b.DB,
	}

	b.Namespace, err = namespaces.Find(b.NamespaceID.Int64)

	return errors.Err(err)
}

func (b *Build) LoadTags() error {
	var err error

	b.Tags, err = b.TagStore().All()

	return errors.Err(err)
}

func (b *Build) LoadStages() error {
	var err error

	b.Stages, err = b.StageStore().All()

	return errors.Err(err)
}

func (b *Build) LoadVariables() error {
	var err error

	if len(b.Variables) == 0 {
		variables := b.BuildVariableStore()

		b.Variables, err = variables.All()

		if err != nil {
			return errors.Err(err)
		}

		if err := variables.LoadVariables(b.Variables); err != nil {
			return errors.Err(err)
		}
	}

	return nil
}

func (b Build) UIEndpoint() string {
	return fmt.Sprintf("/builds/%v", b.ID)
}

func (b *BuildObject) Create() error {
	stmt, err := b.Prepare(`
		INSERT INTO build_objects (build_id, object_id, source)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(b.BuildID, b.ObjectID, b.Source)

	return errors.Err(row.Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt))
}

func (b *BuildVariable) Create() error {
	stmt, err := b.Prepare(`
		INSERT INTO build_variables (build_id, variable_id)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(b.BuildID, b.VariableID)

	return errors.Err(row.Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt))
}
