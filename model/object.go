package model

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type Object struct {
	Model

	UserID    int64       `db:"user_id"`
	Hash      string      `db:"hash"`
	Name      string      `db:"name"`
	Type      string      `db:"type"`
	Size      int64       `db:"size"`
	MD5       []byte      `db:"md5"`
	SHA256    []byte      `db:"sha256"`
	DeletedAt pq.NullTime `db:"deleted_at"`

	User *User
}

type BuildObject struct {
	Model

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

	User *User
}

type BuildObjectStore struct {
	*sqlx.DB

	Build  *Build
	Object *Object
}

func (o Object) AccessibleBy(u *User, a Action) bool {
	if u == nil {
		return false
	}

	return o.UserID == u.ID
}

func (o *Object) BuildObjectStore() BuildObjectStore {
	return BuildObjectStore{
		DB:     o.DB,
		Object: o,
	}
}

func (o *Object) Create() error {
	q := query.Insert(
		query.Table("objects"),
		query.Columns("user_id", "hash", "name", "type", "size", "md5", "sha256"),
		query.Values(o.UserID, o.Hash, o.Name, o.Type, o.Size, o.MD5, o.SHA256),
		query.Returning("id", "created_at", "updated_at"),
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
	q := query.Update(
		query.Table("objects"),
		query.SetRaw("deleted_at", "NOW()"),
		query.WhereEq("id", o.ID),
		query.Returning("deleted_at"),
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
	return o.Model.IsZero() &&
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
	q := query.Insert(
		query.Table("build_objects"),
		query.Columns("build_id", "object_id", "source", "name"),
		query.Values(bo.BuildID, bo.ObjectID, bo.Source, bo.Name),
		query.Returning("id", "created_at", "updated_at"),
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
	q := query.Update(
		query.Table("build_objects"),
		query.Set("placed", bo.Placed),
		query.SetRaw("updated_at", "NOW()"),
		query.WhereEq("id", bo.ID),
		query.Returning("updated_at"),
	)

	stmt, err := bo.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&bo.UpdatedAt))
}

func (os ObjectStore) All(opts ...query.Option) ([]*Object, error) {
	oo := make([]*Object, 0)

	opts = append([]query.Option{query.Columns("*")}, opts...)
	opts = append(opts, ForUser(os.User), query.Table("objects"))

	q := query.Select(opts...)

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

func (os ObjectStore) Index(opts ...query.Option) ([]*Object, error) {
	oo, err := os.All(append(opts, query.WhereIs("deleted_at", "NULL"))...)

	return oo, errors.Err(err)
}

func (os ObjectStore) Find(id int64) (*Object, error) {
	o := &Object{
		Model: Model{
			DB: os.DB,
		},
		User: os.User,
	}

	q := query.Select(
		query.Columns("*"),
		query.Table("objects"),
		query.WhereEq("id", id),
		query.WhereIs("deleted_at", "NULL"),
		ForUser(os.User),
	)

	err := os.Get(o, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return o, errors.Err(err)
}

func (os ObjectStore) FindByName(name string) (*Object, error) {
	o := &Object{
		Model: Model{
			DB: os.DB,
		},
		User: os.User,
	}

	q := query.Select(
		query.Columns("*"),
		query.Table("objects"),
		query.WhereEq("name", name),
		query.WhereIs("deleted_at", "NULL"),
		ForUser(os.User),
	)

	err := os.Get(o, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return o, errors.Err(err)
}

func (os ObjectStore) New() *Object {
	o := &Object{
		Model: Model{
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

func (bos BuildObjectStore) All(opts ...query.Option) ([]*BuildObject, error) {
	boo := make([]*BuildObject, 0)

	opts = append([]query.Option{query.Columns("*")}, opts...)
	opts = append(opts, ForBuild(bos.Build), ForObject(bos.Object), query.Table("build_objects"))

	q := query.Select(opts...)

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
		Model: Model{
			DB: bos.DB,
		},
	}

	boo, err := bos.All(query.Limit(1))

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
		if bo.ObjectID.Valid {
			ids = append(ids, bo.ObjectID.Int64)
		}
	}

	objects := ObjectStore{
		DB: bos.DB,
	}

	oo, err := objects.All(query.WhereIn("id", ids...))

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
		Model: Model{
			DB: bos.DB,
		},
		Build:  bos.Build,
		Object: bos.Object,
	}

	if bos.Build != nil {
		bo.BuildID = bos.Build.ID
	}

	if bos.Object != nil {
		bo.ObjectID = sql.NullInt64{
			Int64: bos.Object.ID,
			Valid: true,
		}
	}

	return bo
}
