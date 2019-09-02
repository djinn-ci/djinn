package model

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"

	"github.com/lib/pq"
)

type Object struct {
	Model

	UserID      int64         `db:"user_id"`
	NamespaceID sql.NullInt64 `db:"namespace_id"`
	Hash        string        `db:"hash"`
	Name        string        `db:"name"`
	Type        string        `db:"type"`
	Size        int64         `db:"size"`
	MD5         []byte        `db:"md5"`
	SHA256      []byte        `db:"sha256"`
	DeletedAt   pq.NullTime   `db:"deleted_at"`

	User      *User
	Namespace *Namespace
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
	Store

	User      *User
	Namespace *Namespace
}

type BuildObjectStore struct {
	Store

	Build  *Build
	Object *Object
}

func buildObjectToInterface(boo []*BuildObject) func(i int) Interface {
	return func(i int) Interface {
		return boo[i]
	}
}

func objectToInterface(oo []*Object) func(i int) Interface {
	return func(i int) Interface {
		return oo[i]
	}
}

func (bo BuildObject) Values() map[string]interface{} {
	return map[string]interface{}{
		"build_id":   bo.BuildID,
		"object_id":  bo.ObjectID,
		"source":     bo.Source,
		"name":       bo.Name,
		"placed":     bo.Placed,
	}
}

func (o *Object) BuildObjectStore() BuildObjectStore {
	return BuildObjectStore{
		Store: Store{
			DB: o.DB,
		},
		Object: o,
	}
}

func (o Object) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":      o.UserID,
		"namespace_id": o.NamespaceID,
		"hash":         o.Hash,
		"name":         o.Name,
		"type":         o.Type,
		"size":         o.Size,
		"md5":          o.MD5,
		"sha256":       o.SHA256,
		"created_at":   o.CreatedAt,
		"updated_at":   o.UpdatedAt,
	}
}

func (o *Object) Destroy() error {
	q := query.Update(
		query.Table(ObjectTable),
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

func (s ObjectStore) All(opts ...query.Option) ([]*Object, error) {
	oo := make([]*Object, 0)

	opts = append(opts, ForCollaborator(s.User), ForNamespace(s.Namespace))

	err := s.Store.All(&oo, ObjectTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, o := range oo {
		o.DB = s.DB
		o.User = s.User
	}

	return oo, errors.Err(err)
}

func (s ObjectStore) interfaceSlice(oo ...*Object) []Interface {
	ii := make([]Interface, len(oo), len(oo))

	for i, o := range oo {
		ii[i] = o
	}

	return ii
}

func (s ObjectStore) Create(oo ...*Object) error {
	models := interfaceSlice(len(oo), objectToInterface(oo))

	return errors.Err(s.Store.Create(ObjectTable, models...))
}

func (s ObjectStore) Delete(oo ...*Object) error {
	models := interfaceSlice(len(oo), objectToInterface(oo))

	return errors.Err(s.Store.Delete(ObjectTable, models...))
}

func (s ObjectStore) Index(opts ...query.Option) ([]*Object, error) {
	oo, err := s.All(append(opts, query.WhereIs("deleted_at", "NULL"))...)

	if err != nil {
		return oo, errors.Err(err)
	}

	if err := s.LoadNamespaces(oo); err != nil {
		return oo, errors.Err(err)
	}

	nn := make([]*Namespace, 0, len(oo))

	for _, o := range oo {
		if o.Namespace != nil {
			nn = append(nn, o.Namespace)
		}
	}

	namespaces := NamespaceStore{
		Store: s.Store,
	}

	err = namespaces.LoadUsers(nn)

	return oo, errors.Err(err)
}

func (s ObjectStore) findBy(col string, val interface{}) (*Object, error) {
	o := &Object{
		Model: Model{
			DB: s.DB,
		},
		User: s.User,
	}

	err := s.FindBy(o, ObjectTable, col, val)

	if err == sql.ErrNoRows {
		err = nil
	}

	return o, errors.Err(err)
}

func (s ObjectStore) Find(id int64) (*Object, error) {
	o, err := s.findBy("id", id)

	return o, errors.Err(err)
}

func (s ObjectStore) FindByName(name string) (*Object, error) {
	o, err := s.findBy("name", name)

	return o, errors.Err(err)
}

func (s ObjectStore) loadNamespace(oo []*Object) func(i int, n *Namespace) {
	return func(i int, n *Namespace) {
		o := oo[i]

		if o.NamespaceID.Int64 == n.ID {
			o.Namespace = n
		}
	}
}

func (s ObjectStore) LoadNamespaces(oo []*Object) error {
	if len(oo) == 0 {
		return nil
	}

	models := interfaceSlice(len(oo), objectToInterface(oo))

	namespaces := NamespaceStore{
		Store: s.Store,
	}

	err := namespaces.Load(mapKey("namespace_id", models), s.loadNamespace(oo))

	return errors.Err(err)
}

func (s ObjectStore) New() *Object {
	o := &Object{
		Model: Model{
			DB: s.DB,
		},
		User: s.User,
	}

	if s.User != nil {
		o.UserID = s.User.ID
	}

	return o
}

func (s ObjectStore) Show(id int64) (*Object, error) {
	o, err := s.Find(id)

	if err != nil {
		return o, errors.Err(err)
	}

	return o, errors.Err(err)
}

func (s BuildObjectStore) All(opts ...query.Option) ([]*BuildObject, error) {
	boo := make([]*BuildObject, 0)

	opts = append(opts, ForBuild(s.Build), ForObject(s.Object))

	err := s.Store.All(&boo, BuildObjectTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, bo := range boo {
		bo.DB = s.DB
		bo.Build = s.Build
		bo.Object = s.Object
	}

	return boo, errors.Err(err)
}

func (s BuildObjectStore) Create(boo ...*BuildObject) error {
	models := interfaceSlice(len(boo), buildObjectToInterface(boo))

	return errors.Err(s.Store.Create(BuildObjectTable, models...))
}

func (s BuildObjectStore) First() (*BuildObject, error) {
	bo := &BuildObject{
		Model: Model{
			DB: s.DB,
		},
	}

	q := query.Select(
		query.Columns("*"),
		query.Table(BuildObjectTable),
	)

	err := s.Store.Get(bo, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return bo, nil
}

func (s BuildObjectStore) LoadObjects(boo []*BuildObject) error {
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
		Store: s.Store,
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

func (s BuildObjectStore) New() *BuildObject {
	bo := &BuildObject{
		Model: Model{
			DB: s.DB,
		},
		Build:  s.Build,
		Object: s.Object,
	}

	if s.Build != nil {
		bo.BuildID = s.Build.ID
	}

	if s.Object != nil {
		bo.ObjectID = sql.NullInt64{
			Int64: s.Object.ID,
			Valid: true,
		}
	}

	return bo
}
