package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type Object struct {
	model

	UserID    int64        `db:"user_id"`
	Hash      string       `db:"hash"`
	Name      string       `db:"name"`
	Type      string       `db:"type"`
	Size      int64        `db:"size"`
	MD5       []byte       `db:"md5"`
	SHA256    []byte       `db:"sha256"`
	DeletedAt *pq.NullTime `db:"deleted_at"`

	User *User
}

type BuildObject struct {
	model

	BuildID     int64         `db:"build_id"`
	ObjectID    sql.NullInt64 `db:"object_id"`
	Source      string        `db:"source"`
	Name        string        `db:"name"`
	Placed      bool          `db:"placed"`

	Build  *Build
	Object *Object
}

type ObjectStore struct {
	*sqlx.DB

	user *User
}

type BuildObjectStore struct {
	*sqlx.DB

	build  *Build
	object *Object
}

func (o *Object) BuildObjectStore() BuildObjectStore {
	return BuildObjectStore{
		DB:     o.DB,
		object: o,
	}
}

func (o *Object) Create() error {
	stmt, err := o.Prepare(`
		INSERT INTO objects (user_id, name, type, size, md5, sha256)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(o.UserID, o.Name, o.Type, o.Size, o.MD5, o.SHA256)

	return errors.Err(row.Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt))
}

func (o *Object) IsZero() bool {
	return o.model.IsZero() &&
           o.UserID == 0 &&
           o.Name == "" &&
           o.Type == "" &&
           o.Size == 0 &&
           len(o.MD5) == 0 &&
           len(o.SHA256) == 0 &&
           o.DeletedAt == nil
}

func (bo *BuildObject) Create() error {
	stmt, err := bo.Prepare(`
		INSERT INTO build_objects (build_id, object_id, source, name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(bo.BuildID, bo.ObjectID, bo.Source, bo.Name)

	return errors.Err(row.Scan(&bo.ID, &bo.CreatedAt, &bo.UpdatedAt))
}

func (bo *BuildObject) Update() error {
	stmt, err := bo.Prepare(`
		UPDATE build_objects
		SET placed = $1 WHERE id = $2
		RETURNING updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(bo.Placed, bo.ID)

	return errors.Err(row.Scan(&bo.UpdatedAt))
}

func (os ObjectStore) Find(id int64) (*Object, error) {
	o := &Object{
		model: model{
			DB: os.DB,
		},
	}

	query := "SELECT * FROM objects WHERE id = $1"
	args := []interface{}{id}

	if os.user != nil {
		query += " AND user_id = $2"
		args = append(args, os.user.ID)

		o.User = os.user
	}

	err := os.Get(o, query, args...)

	if err == sql.ErrNoRows {
		err = nil

		o.CreatedAt = nil
		o.UpdatedAt = nil
		o.DeletedAt = nil
	}

	return o, errors.Err(err)
}

func (os ObjectStore) FindByName(name string) (*Object, error) {
	o := &Object{
		model: model{
			DB: os.DB,
		},
		User: os.user,
	}

	query := "SELECT * FROM objects WHERE name = $1"
	args := []interface{}{name}

	if os.user != nil {
		query += " AND user_id = $2"
		args = append(args, os.user.ID)
	}

	err := os.Get(o, query, args...)

	if err == sql.ErrNoRows {
		err = nil

		o.CreatedAt = nil
		o.UpdatedAt = nil
		o.DeletedAt = nil
	}

	return o, errors.Err(err)
}

func (os ObjectStore) In(ids ...int64) ([]*Object, error) {
	oo := make([]*Object, 0)

	if len(ids) == 0 {
		return oo, nil
	}

	query, args, err := sqlx.In("SELECT * FROM objects WHERE id IN (?)", ids)

	if err != nil {
		return oo, errors.Err(err)
	}

	err = os.Select(&oo, os.Rebind(query), args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, o := range oo {
		o.DB = os.DB
	}

	return oo, errors.Err(err)
}

func (os ObjectStore) New() *Object {
	o := &Object{
		model: model{
			DB: os.DB,
		},
		User: os.user,
	}

	if os.user != nil {
		o.UserID = os.user.ID
	}

	return o
}

func (bos BuildObjectStore) All() ([]*BuildObject, error) {
	bo := make([]*BuildObject, 0)

	query := "SELECT * FROM build_objects"
	args := []interface{}{}

	if bos.build != nil {
		query += " WHERE build_id = $1"
		args = append(args, bos.build.ID)
	}

	if bos.object != nil {
		if bos.build != nil {
			query += " AND object_id = $2"
		} else {
			query += " WHERE object_id = $1"
		}

		args = append(args, bos.object.ID)
	}

	err := bos.Select(&bo, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return bo, errors.Err(err)
}

func (bos BuildObjectStore) First() (*BuildObject, error) {
	bo := &BuildObject{
		model: model{
			DB: bos.DB,
		},
	}

	query := "SELECT * FROM build_objects"
	args := []interface{}{}

	if bos.build != nil {
		query += " WHERE build_id = $1"
		args = append(args, bos.build.ID)
	}

	if bos.object != nil {
		if bos.build != nil {
			query += " AND object_id = $2"
		} else {
			query += " WHERE object_id = $1"
		}

		args = append(args, bos.object.ID)
	}

	err := bos.Get(&bo, query, args...)

	if err == sql.ErrNoRows {
		err = nil

		bo.CreatedAt = nil
		bo.UpdatedAt = nil
	}

	return bo, errors.Err(err)
}

func (bos BuildObjectStore) LoadObjects(boo []*BuildObject) error {
	if len(boo) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(boo))

	for _, bo := range boo {
		if bo.ObjectID.Valid {
			ids = append(ids, bo.ObjectID.Int64)
		}
	}

	objects := ObjectStore{
		DB: bos.DB,
	}

	oo, err := objects.In(ids...)

	if err != nil {
		return errors.Err(err)
	}

	for _, bo := range boo {
		for _, o := range oo {
			if bo.ObjectID.Int64 == o.ID {
				bo.Object = o
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
		Build:  bos.build,
		Object: bos.object,
	}

	if bos.build != nil {
		bo.BuildID = bos.build.ID
	}

	if bos.object != nil {
		bo.ObjectID = sql.NullInt64{
			Int64: bos.object.ID,
			Valid: true,
		}
	}

	return bo
}
