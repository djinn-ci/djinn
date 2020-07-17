package build

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/object"
	"github.com/andrewpillar/thrall/runner"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Object is the type that represents a file that has been placed into a build
// environment for a given build. This contains metadata about the file that
// was placed.
type Object struct {
	ID        int64         `db:"id"`
	BuildID   int64         `db:"build_id"`
	ObjectID  sql.NullInt64 `db:"object_id"`
	Source    string        `db:"source"`
	Name      string        `db:"name"`
	Placed    bool          `db:"placed"`
	CreatedAt time.Time     `db:"created_at"`

	Build  *Build         `db:"-"`
	Object *object.Object `db:"-"`
}

// ObjectStore is the type for creating and modifying Object models in the
// database. The ObjectStore type can have an underlying runner.Placer
// implementation that can allow for it to be used for placing objects
// into a build environment.
type ObjectStore struct {
	database.Store

	placer runner.Placer

	// Build is the bound Build model. If not nil this will bind the Build
	// model to any Object models that are created. If not nil this will
	// append a WHERE clause on the build_id column for all SELECT queries
	// performed.
	Build *Build

	// Object is the bound object.Object model. If not nil this will bind the
	// object.Object model to any Object models that are created. If not nil
	// this will append a WHERE clause on the object_id column for all SELECT
	// queries performed.
	Object *object.Object
}

var (
	_ database.Model  = (*Object)(nil)
	_ database.Binder = (*ObjectStore)(nil)
	_ database.Loader = (*ObjectStore)(nil)
	_ runner.Placer   = (*ObjectStore)(nil)

	objectTable = "build_objects"
)

// NewObjectStore returns a new ObjectStore for querying the build_objects
// table. Each model passed to this function will be bound to the returned
// ObjectStore.
func NewObjectStore(db *sqlx.DB, mm ...database.Model) *ObjectStore {
	s := &ObjectStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// NewObjectStoreWithCollector returns a new ObjectStore with the given
// runner.Placer to use for object placement.
func NewObjectStoreWithPlacer(db *sqlx.DB, p runner.Placer, mm ...database.Model) *ObjectStore {
	s := NewObjectStore(db, mm...)
	s.placer = p
	return s
}

// ObjectModel is called along with database.ModelSlice to convert the given slice of
// Object models to a slice of database.Model interfaces.
func ObjectModel(oo []*Object) func(int) database.Model {
	return func(i int) database.Model {
		return oo[i]
	}
}

// SelectObject returns SELECT query that will select the given column from the
// build_objects table with the given query options applied.
func SelectObject(col string, opts ...query.Option) query.Query {
	return query.Select(append([]query.Option{
		query.Columns(col),
		query.From(objectTable),
	}, opts...)...)
}

// Bind implements the database.Binder interface. This will only bind the models
// if they are pointers to either Build or object.Object.
func (o *Object) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			o.Build = m.(*Build)
		case *object.Object:
			o.Object = m.(*object.Object)
		}
	}
}

// SetPrimary implements the database.Model interface.
func (o *Object) SetPrimary(id int64) { o.ID = id }

// Primary implements the database.Model interface.
func (o *Object) Primary() (string, int64) { return "id", o.ID }

// Endpoint implements the database.Model interface. This returns an empty
// string.
func (o *Object) Endpoint(_ ...string) string { return "" }

// IsZero implements the database.Model interface.
func (o *Object) IsZero() bool {
	return o == nil || o.ID == 0 &&
		o.BuildID == 0 &&
		!o.ObjectID.Valid &&
		o.Source == "" &&
		o.Name == "" &&
		!o.Placed &&
		o.CreatedAt == time.Time{}
}

// JSON implements the database.Model interface. This will return a map with the
// current Object's values under each key. If any of the Build, or Object bound
// models exist on the Object, then the JSON representation of these models will
// be in the returned map, under the build, and object keys respectively.
func (o *Object) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":       o.ID,
		"build_id": o.BuildID,
		"source":   o.Source,
		"name":     o.Name,
		"type":     nil,
		"md5":      nil,
		"sha256":   nil,
		"placed":   o.Placed,
	}

	if !o.Build.IsZero() {
		json["build"] = o.Build.JSON(addr)
	}

	if !o.Object.IsZero() {
		json["type"] = o.Object.Type
		json["md5"] = fmt.Sprintf("%x", o.Object.MD5)
		json["sha256"] = fmt.Sprintf("%x", o.Object.SHA256)
		json["object_url"] = addr + o.Object.Endpoint()
	}
	return json
}

// Values implements the database.Model interface. This will return a map with
// the following values, build_id, object_id, source, name, and placed.
func (o *Object) Values() map[string]interface{} {
	return map[string]interface{}{
		"build_id":  o.BuildID,
		"object_id": o.ObjectID,
		"source":    o.Source,
		"name":      o.Name,
		"placed":    o.Placed,
	}
}

// New returns a new Object binding any non-nil models to it from the current
// ObjectStore.
func (s *ObjectStore) New() *Object {
	o := &Object{
		Build:  s.Build,
		Object: s.Object,
	}

	if s.Build != nil {
		_, id := s.Build.Primary()
		o.BuildID = id
	}

	if s.Object != nil {
		_, id := s.Object.Primary()
		o.ObjectID = sql.NullInt64{
			Int64: id,
			Valid: true,
		}
	}
	return o
}

func (s *ObjectStore) Create(objectId int64, name, dst string) (*Object, error) {
	o := s.New()
	o.ObjectID = sql.NullInt64{
		Int64: objectId,
		Valid: objectId > 0,
	}
	o.Source = name
	o.Name = dst

	err := s.Store.Create(objectTable, o)
	return o, errors.Err(err)
}

// All returns a slice of Object models, applying each query.Option that is
// given.
func (s *ObjectStore) All(opts ...query.Option) ([]*Object, error) {
	oo := make([]*Object, 0)

	opts = append([]query.Option{
		database.Where(s.Build, "build_id"),
		database.Where(s.Object, "object_id"),
	}, opts...)

	err := s.Store.All(&oo, objectTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, o := range oo {
		o.Build = s.Build
		o.Object = s.Object
	}
	return oo, errors.Err(err)
}

// Get returns a single Object model, applying each query.Option that is given.
func (s *ObjectStore) Get(opts ...query.Option) (*Object, error) {
	o := &Object{
		Build:  s.Build,
		Object: s.Object,
	}

	opts = append([]query.Option{
		database.Where(s.Build, "build_id"),
		database.Where(s.Object, "object_id"),
	}, opts...)

	err := s.Store.Get(o, objectTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return o, errors.Err(err)
}

// Bind implements the database.Binder interface. This will only bind the models
// if they are pointers to either Build or object.Object.
func (s *ObjectStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			s.Build = m.(*Build)
		case *object.Object:
			s.Object = m.(*object.Object)
		}
	}
}

// Load loads in a slice of Object models where the given key is in the list of
// given vals. Each database is loaded individually via a call to the given load
// callback. This method calls ObjectStore.All under the hood, so any bound
// models will impact the models being loaded.
func (s *ObjectStore) Load(key string, vals []interface{}, load database.LoaderFunc) error {
	oo, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, o := range oo {
			load(i, o)
		}
	}
	return nil
}

func (s *ObjectStore) getObjectToPlace(name string) (*Object, error) {
	if s.placer == nil {
		return nil, errors.New("cannot place object: nil placer")
	}

	o, err := s.Get(query.Where("source", "=", name))

	if err != nil {
		return o, errors.Err(err)
	}

	if o.IsZero() {
		return o, errors.New("cannot find object: " + name)
	}

	o.Object, err = object.NewStore(s.DB).Get(query.Where("id", "=", o.ObjectID))

	if err != nil {
		return o, errors.Err(err)
	}

	if o.Object.IsZero() {
		return o, errors.New("cannot find object: " + name)
	}
	return o, nil
}

// Place looks up the Object by the given name, and updates it once object
// placement has been done via the underlying runner.Placer on the store.
func (s *ObjectStore) Place(name string, w io.Writer) (int64, error) {
	o, err := s.getObjectToPlace(name)

	if err != nil {
		return 0, errors.Err(err)
	}

	n, errPlace := s.placer.Place(o.Object.Hash, w)

	if errors.Cause(errPlace) == io.EOF {
		errPlace = nil
	}

	o.Placed = errPlace == nil

	q := query.Update(
		query.Table(objectTable),
		query.Set("placed", o.Placed),
		query.Where("id", "=", o.ID),
	)

	_, err = s.DB.Exec(q.Build(), q.Args()...)
	return n, errors.Err(err)
}

// Stat returns the os.FileInfo of the Object by the given name. This uses the
// underlying runner.Placer on the store.
func (s *ObjectStore) Stat(name string) (os.FileInfo, error) {
	o, err := s.getObjectToPlace(name)

	if err != nil {
		return nil, errors.Err(err)
	}

	info, err := s.placer.Stat(o.Object.Hash)
	return info, errors.Err(err)
}
