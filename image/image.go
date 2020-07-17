// Package image provides the database.Model implementation for the Image entity.
package image

import (
	"context"
	"database/sql"
	"io"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/block"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Image is the type that represents an image that has been uploaded by a user.
type Image struct {
	ID          int64         `db:"id"`
	UserID      int64         `db:"user_id"`
	NamespaceID sql.NullInt64 `db:"namespace_id"`
	Driver      driver.Type   `db:"driver"`
	Hash        string        `db:"hash"`
	Name        string        `db:"name"`
	CreatedAt   time.Time     `db:"created_at"`

	User      *user.User           `db:"-"`
	Namespace *namespace.Namespace `db:"-"`
}

// Store is the type for creating and modifying Image models in the
// database. The Store type can have an underlying block.Store implementation
// that is used for storing the contents of an image.
type Store struct {
	database.Store

	blockStore block.Store

	// User is the bound user.User model. If not nil this will bind the
	// user.User model to any Image models that are created. If not nil this
	// will append a WHERE clause on the user_id column for all SELECT queries
	// performed.
	User *user.User

	// Namespace is the bound namespace.Namespace model. If not nil this will
	// bind the namespace.Namespace model to any Image models that are created.
	// If not nil this will append a WHERE clause on the namespace_id column for
	// all SELECT queries performed.
	Namespace *namespace.Namespace
}

var (
	_ database.Model  = (*Image)(nil)
	_ database.Loader = (*Store)(nil)
	_ database.Binder = (*Store)(nil)

	table     = "images"
	relations = map[string]database.RelationFunc{
		"namespace": database.Relation("namespace_id", "id"),
	}
)

// NewStore returns a new Store for querying the images table. Each model passed
// to this function will be bound to the returned Store.
func NewStore(db *sqlx.DB, mm ...database.Model) *Store {
	s := &Store{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// NewStoreWithBlockStore is functionally the same as NewStore, however it sets
// the block.Store to use on the returned Store. This will allow for an image
// file to be stored.
func NewStoreWithBlockStore(db *sqlx.DB, blockStore block.Store, mm ...database.Model) *Store {
	s := NewStore(db, mm...)
	s.blockStore = blockStore
	return s
}

// FromContext returns the Image model from the given context, if any.
func FromContext(ctx context.Context) (*Image, bool) {
	i, ok := ctx.Value("image").(*Image)
	return i, ok
}

// LoadRelations loads all of the available relations for the given Image models
// using the given loaders available.
func LoadRelations(loaders *database.Loaders, ii ...*Image) error {
	mm := database.ModelSlice(len(ii), Model(ii))
	return database.LoadRelations(relations, loaders, mm...)
}

// Model is called along with database.ModelSlice to convert the given slice of
// Image models to a slice of database.Model interfaces.
func Model(ii []*Image) func(int) database.Model {
	return func(i int) database.Model {
		return ii[i]
	}
}

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either user.User or namespace.Namespace.
func (i *Image) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			i.User = m.(*user.User)
		case *namespace.Namespace:
			i.Namespace = m.(*namespace.Namespace)
		}
	}
}

// SetPrimary implements the database.Model interface.
func (i *Image) SetPrimary(id int64) { i.ID = id }

// Primary implements the database.Model interface.
func (i *Image) Primary() (string, int64) { return "id", i.ID }

// IsZero implements the database.Model interface.
func (i *Image) IsZero() bool {
	return i == nil || i.ID == 0 &&
		i.UserID == 0 &&
		!i.NamespaceID.Valid &&
		i.Driver == driver.Type(0) &&
		i.Hash == "" &&
		i.Name == "" &&
		i.CreatedAt == time.Time{}
}

// JSON implements the database.Model interface. This will return a map with the
// current Image values under each key. If any of the User, or Namespace bound
// models exist on the Image, then the JSON representation of these models
// will be in the returned map, under the user, and namespace keys respectively.
func (i *Image) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":           i.ID,
		"user_id":      i.UserID,
		"namespace_id": nil,
		"name":         i.Name,
		"created_at":   i.CreatedAt.Format(time.RFC3339),
		"url":          addr + i.Endpoint(),
	}

	if i.NamespaceID.Valid {
		json["namespace_id"] = i.NamespaceID.Int64
	}

	for name, m := range map[string]database.Model{
		"user":      i.User,
		"namespace": i.Namespace,
	} {
		if !m.IsZero() {
			json[name] = m.JSON(addr)
		}
	}
	return json
}

// Endpoint implements the database.Model interface. This will return the
// endpoint to the current Image model.
func (i *Image) Endpoint(uri ...string) string {
	if len(uri) > 0 {
		return "/images/" + strconv.FormatInt(i.ID, 10) + "/" + strings.Join(uri, "/")
	}
	return "/images/" + strconv.FormatInt(i.ID, 10)
}

// Values implements the database.Model interface. This will return a map with
// the following values, user_id, namespace_id, driver, hash, name, and
// created_at.
func (i *Image) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":      i.UserID,
		"namespace_id": i.NamespaceID,
		"driver":       i.Driver,
		"hash":         i.Hash,
		"name":         i.Name,
		"created_at":   i.CreatedAt,
	}
}

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either user.User or namespace.Namespace.
func (s *Store) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		case *namespace.Namespace:
			s.Namespace = m.(*namespace.Namespace)
		}
	}
}

// Create creates a new image with the given name for the given driver.Type.
// The given io.Reader is used to copy the contents of the image to the
// underlying block.Store. It is expected for the Store to have a block.Store
// set on it, otherwise it will error.
func (s *Store) Create(hash, name string, t driver.Type, r io.Reader) (*Image, error) {
	if s.blockStore == nil {
		return nil, errors.New("nil block store")
	}

	dst, err := s.blockStore.Create(filepath.Join(t.String(), hash))

	if err != nil {
		return nil, errors.Err(err)
	}

	defer dst.Close()

	if _, err := io.Copy(dst, r); err != nil {
		return nil, errors.Err(err)
	}

	i := s.New()
	i.Driver = t
	i.Hash = hash
	i.Name = name

	err = s.Store.Create(table, i)
	return i, errors.Err(err)
}

// Delete deletes the given Image from the database, and removes the underlying
// image file. It is expected for the Store to have a block.Store set on it,
// otherwise it will error.
func (s *Store) Delete(id int64, t driver.Type, hash string) error {
	if s.blockStore == nil {
		return errors.New("nil block store")
	}

	if err := s.Store.Delete(table, &Image{ID: id}); err != nil {
		return errors.Err(err)
	}
	return errors.Err(s.blockStore.Remove(filepath.Join(t.String(), hash)))
}

// All returns a slice of Image models, applying each query.Option that is
// given.
func (s *Store) All(opts ...query.Option) ([]*Image, error) {
	ii := make([]*Image, 0)

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		database.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.All(&ii, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, i := range ii {
		i.User = s.User
		i.Namespace = s.Namespace
	}
	return ii, errors.Err(err)
}

// Index returns the paginated results from the images table depending on the
// values that are present in url.Values. Detailed below are the values that
// are used from the given url.Values,
//
// search - This applies the database.Search query.Option using the value of
// name
func (s *Store) Index(vals url.Values, opts ...query.Option) ([]*Image, database.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		database.Search("name", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, opts...)

	if err != nil {
		return []*Image{}, paginator, errors.Err(err)
	}

	ii, err := s.All(append(
		opts,
		query.OrderAsc("name"),
		query.Limit(database.PageLimit),
		query.Offset(paginator.Offset),
	)...)
	return ii, paginator, errors.Err(err)
}

// Load loads in a slice of Image models where the given key is in the list
// of given vals. Each database is loaded individually via a call to the given
// load callback. This method calls Store.All under the hood, so any
// bound models will impact the models being loaded.
func (s *Store) Load(key string, vals []interface{}, fn database.LoaderFunc) error {
	ii, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for j := range vals {
		for _, i := range ii {
			fn(j, i)
		}
	}
	return nil
}

// Get returns a single Image model, applying each query.Option that is given.
func (s *Store) Get(opts ...query.Option) (*Image, error) {
	i := &Image{
		User:      s.User,
		Namespace: s.Namespace,
	}

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		database.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.Get(i, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return i, errors.Err(err)
}

// New returns a new Image binding any non-nil models to it from the current
// Store.
func (s *Store) New() *Image {
	i := &Image{
		User:      s.User,
		Namespace: s.Namespace,
	}

	if s.User != nil {
		_, id := s.User.Primary()
		i.UserID = id
	}

	if s.Namespace != nil {
		_, id := s.Namespace.Primary()
		i.NamespaceID = sql.NullInt64{
			Int64: id,
			Valid: true,
		}
	}
	return i
}

// Paginate returns the database.Paginator for the images table for the given
// page.
func (s *Store) Paginate(page int64, opts ...query.Option) (database.Paginator, error) {
	paginator, err := s.Store.Paginate(table, page, opts...)
	return paginator, errors.Err(err)
}
