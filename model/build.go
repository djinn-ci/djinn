package model

import (
	"fmt"
	"database/sql"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type Build struct {
	model

	UserID      int64          `db:"user_id"`
	NamespaceID sql.NullInt64  `db:"namespace_id"`
	Manifest    string         `db:"manifest"`
	Status      Status         `db:"status"`
	Output      sql.NullString `db:"output"`
	StartedAt   *pq.NullTime   `db:"started_at"`
	FinishedAt  *pq.NullTime   `db:"finished_at"`

	User      *User
	Namespace *Namespace
	Stages    []*Stage
	Tags      []*Tag
	Variables []*BuildVariable
}

type BuildObject struct {
	model

	BuildID  int64  `db:"build_id"`
	ObjectID int64  `db:"object_id"`
	Source   string `db:"source"`
	Placed   bool   `db:"placed"`

	Build  *Build
	Object *Object
}

type BuildVariable struct {
	model

	BuildID    int64 `db:"build_id"`
	VariableID int64 `db:"variable_id"`

	Build    *Build
	Variable *Variable
}

type BuildStore struct {
	*Store

	user      *User
	namespace *Namespace
}

type BuildObjectStore struct {
	*Store

	build  *Build
	object *Object
}

type BuildVariableStore struct {
	*Store

	build    *Build
	variable *Variable
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

	if len(bb) == 0 {
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

func (bs *BuildStore) LoadRelations(bb []*Build) error {
	if len(bb) == 0 {
		return nil
	}

	namespaceIds := make([]int64, 0, len(bb))
	buildIds := make([]int64, len(bb), len(bb))
	userIds := make([]int64, len(bb), len(bb))

	for i, b := range bb {
		if b.NamespaceID.Valid {
			namespaceIds = append(namespaceIds, b.NamespaceID.Int64)
		}

		buildIds[i] = b.ID
		userIds[i] = b.UserID
	}

	namespaces := NamespaceStore{
		Store: &Store{
			DB: bs.DB,
		},
	}

	nn, err := namespaces.In(namespaceIds...)

	if err != nil {
		return errors.Err(err)
	}

	if err := namespaces.LoadRelations(nn); err != nil {
		return errors.Err(err)
	}

	tags := TagStore{
		Store: &Store{
			DB: bs.DB,
		},
	}

	tt, err := tags.InBuildID(buildIds...)

	if err != nil {
		return errors.Err(err)
	}

	users := UserStore{
		Store: &Store{
			DB: bs.DB,
		},
	}

	uu, err := users.In(userIds...)

	if err != nil {
		return errors.Err(err)
	}

	for _, b := range bb {
		for _, t := range tt {
			if t.BuildID == b.ID {
				b.Tags = append(b.Tags, t)
			}
		}

		for _, n := range nn {
			if n.ID == b.NamespaceID.Int64 && b.Namespace == nil {
				b.Namespace = n
			}
		}

		for _, u := range uu {
			if u.ID == b.UserID && b.User == nil {
				b.User = u
			}
		}
	}

	return nil
}

func (bos BuildObjectStore) New() *BuildObject {
	bo := &BuildObject{
		model: model{
			DB: bos.DB,
		},
	}

	if bos.build != nil {
		bo.BuildID = bos.build.ID
		bo.Build = bos.build
	}

	if bos.object != nil {
		bo.ObjectID = bos.object.ID
		bo.Object = bos.object
	}

	return bo
}

func (bvs BuildVariableStore) New() *BuildVariable {
	bv := &BuildVariable{
		model: model{
			DB: bvs.DB,
		},
		Build: bvs.build,
	}

	if bvs.build != nil {
		bv.BuildID = bvs.build.ID
	}

	return bv
}

func (bvs BuildVariableStore) All() ([]*BuildVariable, error) {
	vv := make([]*BuildVariable, 0)

	query := "SELECT * FROM build_variables"
	args := []interface{}{}

	if bvs.build != nil {
		query += " WHERE build_id = $1"
		args = append(args, bvs.build.ID)
	}

	err := bvs.Select(&vv, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, v := range vv {
		v.DB = bvs.DB
		v.Build = bvs.build
	}

	return vv, errors.Err(err)
}

func (b *Build) ArtifactStore() ArtifactStore {
	return ArtifactStore{
		Store: &Store{
			DB: b.DB,
		},
		build: b,
	}
}

func (b *Build) DriverStore() DriverStore {
	return DriverStore{
		Store: &Store{
			DB: b.DB,
		},
		build: b,
	}
}

func (b *Build) TagStore() TagStore {
	return TagStore{
		Store: &Store{
			DB: b.DB,
		},
		user:  b.User,
		build: b,
	}
}

func (b *Build) StageStore() StageStore {
	return StageStore{
		Store: &Store{
			DB: b.DB,
		},
		build: b,
	}
}

func (b *Build) JobStore() JobStore {
	return JobStore{
		Store: &Store{
			DB: b.DB,
		},
		build: b,
	}
}

func (b *Build) BuildObjectStore() BuildObjectStore {
	return BuildObjectStore{
		Store: &Store{
			DB: b.DB,
		},
		build: b,
	}
}

func (b *Build) BuildVariableStore() BuildVariableStore {
	return BuildVariableStore{
		Store: &Store{
			DB: b.DB,
		},
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
           b.Status == Status(0) &&
           !b.Output.Valid &&
           b.StartedAt == nil &&
           b.FinishedAt == nil &&
           b.CreatedAt == nil &&
           b.UpdatedAt == nil
}

func (b *Build) LoadRelations() error {
	var err error

	if b.User == nil {
		users := UserStore{
			Store: &Store{
				DB: b.DB,
			},
		}

		b.User, err = users.Find(b.UserID)

		if err != nil {
			return errors.Err(err)
		}
	}

	if b.Namespace == nil && b.NamespaceID.Valid {
		namespaces := NamespaceStore{
			Store: &Store{
				DB: b.DB,
			},
		}

		b.Namespace, err = namespaces.Find(b.NamespaceID.Int64)

		if err != nil {
			return errors.Err(err)
		}
	}

	if len(b.Tags) == 0 {
		b.Tags, err = b.TagStore().All()

		if err != nil {
			return errors.Err(err)
		}
	}

	if len(b.Stages) == 0 {
		b.Stages, err = b.StageStore().All()

		return errors.Err(err)
	}

	return nil
}

func (bvs BuildVariableStore) LoadVariables(bvv []*BuildVariable) error {
	if len(bvv) == 0 {
		return nil
	}

	variables := VariableStore{
		Store: &Store{
			DB: bvs.DB,
		},
	}

	ids := make([]int64, len(bvv), len(bvv))

	for i, bv := range bvv {
		ids[i] = bv.VariableID
	}

	vv, err := variables.In(ids...)

	if err != nil {
		return errors.Err(err)
	}

	for _, v := range vv {
		for _, bv := range bvv {
			if v.ID == bv.VariableID {
				bv.Variable = v
			}
		}
	}

	return nil
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
