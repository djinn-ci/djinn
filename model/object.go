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
	Name      string       `db:"name"`
	Filename  string       `db:"filename"`
	Type      string       `db:"type"`
	Size      int64        `db:"size"`
	MD5       []byte       `db:"md5"`
	SHA256    []byte       `db:"sha256"`
	DeletedAt *pq.NullTime `db:"deleted_at"`

	User *User
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
		INSERT INTO objects (user_id, name, filename, type, size, md5, sha256)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(o.UserID, o.Name, o.Filename, o.Type, o.Size, o.MD5, o.SHA256)

	return errors.Err(row.Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt))
}

func (o *Object) IsZero() bool {
	return o.model.IsZero() &&
           o.UserID == 0 &&
           o.Name == "" &&
           o.Filename == "" &&
           o.Type == "" &&
           o.Size == 0 &&
           len(o.MD5) == 0 &&
           len(o.SHA256) == 0 &&
           o.DeletedAt == nil
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
		bo.ObjectID = bos.object.ID
	}

	return bo
}
