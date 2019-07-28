package model

import (
	"database/sql"
	"fmt"
	"strings"

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

	BuildID     int64  `db:"build_id"`
	ObjectID    int64  `db:"object_id"`
	Source      string `db:"source"`
	Name        string `db:"name"`
	Placed      bool   `db:"placed"`

	Build  *Build
	Object *Object
}

type ObjectStore struct {
	*sqlx.DB

	User *User
}

type BuildObjectStore struct {
	*sqlx.DB

	Build  *Build
	Object *Object
}

func (o *Object) BuildObjectStore() BuildObjectStore {
	return BuildObjectStore{
		DB:     o.DB,
		Object: o,
	}
}

func (o *Object) Create() error {
	q := Insert(
		Table("objects"),
		Columns("user_id", "hash", "name", "type", "size", "md5", "sha256"),
		Values(o.UserID, o.Hash, o.Name, o.Type, o.Size, o.MD5, o.SHA256),
		Returning("id", "created_at", "updated_at"),
	)

	stmt, err := o.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt))
}

func (o *Object) Destroy() error {
	q := Update(
		Table("objects"),
		SetRaw("deleted_at", "NOW()"),
		WhereEq("id", o.ID),
		Returning("deleted_at"),
	)

	stmt, err := o.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&o.DeletedAt))
}

func (o *Object) IsZero() bool {
	return o.model.IsZero() &&
           o.UserID == 0 &&
           o.Name == "" &&
           o.Type == "" &&
           o.Size == 0 &&
           len(o.MD5) == 0 &&
           len(o.SHA256) == 0 &&
           !o.DeletedAt.Valid
}

func (o Object) UIEndpoint(uri ...string) string {
	endpoint := fmt.Sprintf("/objects/%v", o.ID)

	if len(uri) > 0 {
		endpoint = fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}

	return endpoint
}

func (bo *BuildObject) Create() error {
	q := Insert(
		Table("build_objects"),
		Columns("build_id", "object_id", "source", "name"),
		Values(bo.BuildID, bo.ObjectID, bo.Source, bo.Name),
		Returning("id", "created_at", "updated_at"),
	)

	stmt, err := bo.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&bo.ID, &bo.CreatedAt, &bo.UpdatedAt))
}

func (bo *BuildObject) Update() error {
	q := Update(
		Table("build_objects"),
		Set("placed", bo.Placed),
		SetRaw("updated_at", "NOW()"),
		WhereEq("id", bo.ID),
		Returning("updated_at"),
	)

	stmt, err := bo.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&bo.UpdatedAt))
}

func (os ObjectStore) All(opts ...Option) ([]*Object, error) {
	oo := make([]*Object, 0)

	opts = append([]Option{Columns("*")}, opts...)

	q := Select(append(opts, ForUser(os.User), Table("objects"))...)

	err := os.Select(&oo, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, o := range oo {
		o.DB = os.DB
		o.User = os.User
	}

	return oo, errors.Err(err)
}

func (os ObjectStore) Index(opts ...Option) ([]*Object, error) {
	oo, err := os.All(append(opts, WhereIs("deleted_at", "NULL"))...)

	return oo, errors.Err(err)
}

func (os ObjectStore) Find(id int64) (*Object, error) {
	o := &Object{
		model: model{
			DB: os.DB,
		},
		User: os.User,
	}

	q := Select(
		Columns("*"),
		Table("objects"),
		WhereEq("id", id),
		WhereIs("deleted_at", "NULL"),
		ForUser(os.User),
	)

	err := os.Get(o, q.Build(), q.Args()...)

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
		User: os.User,
	}

	q := Select(
		Columns("*"),
		Table("objects"),
		WhereEq("name", name),
		WhereIs("deleted_at", "NULL"),
		ForUser(os.User),
	)

	err := os.Get(o, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil

		o.CreatedAt = nil
		o.UpdatedAt = nil
	}

	return o, errors.Err(err)
}

func (os ObjectStore) New() *Object {
	o := &Object{
		model: model{
			DB: os.DB,
		},
		User: os.User,
	}

	if os.User != nil {
		o.UserID = os.User.ID
	}

	return o
}

func (os ObjectStore) Show(id int64) (*Object, error) {
	o, err := os.Find(id)

	if err != nil {
		return o, errors.Err(err)
	}



	return o, errors.Err(err)
}

func (bos BuildObjectStore) All(opts ...Option) ([]*BuildObject, error) {
	boo := make([]*BuildObject, 0)

	opts = append([]Option{Columns("*")}, opts...)

	q := Select(append(opts, ForBuild(bos.Build), ForObject(bos.Object), Table("build_objects"))...)

	err := bos.Select(&boo, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, o := range boo {
		o.DB = bos.DB
		o.Build = bos.Build
		o.Object = bos.Object
	}

	return boo, errors.Err(err)
}

func (bos BuildObjectStore) First() (*BuildObject, error) {
	empty := &BuildObject{
		model: model{
			DB: bos.DB,
		},
	}

	boo, err := bos.All(Limit(1))

	if err != nil {
		return empty, errors.Err(err)
	}

	if len(boo) >= 1 {
		return boo[0], nil
	}

	return empty, nil
}

func (bos BuildObjectStore) LoadObjects(boo []*BuildObject) error {
	if len(boo) == 0 {
		return nil
	}

	ids := make([]interface{}, 0, len(boo))

	for _, bo := range boo {
		ids = append(ids, bo.ObjectID)
	}

	objects := ObjectStore{
		DB: bos.DB,
	}

	oo, err := objects.All(WhereIn("id", ids...))

	if err != nil {
		return errors.Err(err)
	}

	for _, bo := range boo {
		for _, o := range oo {
			if bo.ObjectID == o.ID {
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
		Build:  bos.Build,
		Object: bos.Object,
	}

	if bos.Build != nil {
		bo.BuildID = bos.Build.ID
	}

	if bos.Object != nil {
		bo.ObjectID = bos.Object.ID
	}

	return bo
}
