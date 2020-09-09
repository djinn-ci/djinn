// Package key providers the database.Model implementation for the Key entity.
package key

import (
	"context"
	"database/sql"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Key is the type that represents an SSH key that can be placed in the build
// environment.
type Key struct {
	ID          int64         `db:"id"`
	UserID      int64         `db:"user_id"`
	NamespaceID sql.NullInt64 `db:"namespace_id"`
	Name        string        `db:"name"`
	Key         []byte        `db:"key"`
	Config      string        `db:"config"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`

	User      *user.User           `db:"-"`
	Namespace *namespace.Namespace `db:"-"`
}

// Store is the type for creating and modifying Key models in the database. The
// Store type can have an underlying crypto.Block for encrypting the SSH keys
// that are stored.
type Store struct {
	database.Store

	block *crypto.Block

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
	_ database.Model  = (*Key)(nil)
	_ database.Binder = (*Store)(nil)
	_ database.Loader = (*Store)(nil)

	table     = "keys"
	relations = map[string]database.RelationFunc{
		"user":      database.Relation("user_id", "id"),
		"namespace": database.Relation("namespace_id", "id"),
	}
)

// NewStore returns a new Store for querying the keys table. Each model passed
// to this function will be bound to the returned Store.
func NewStore(db *sqlx.DB, mm ...database.Model) *Store {
	s := &Store{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// NewStoreWithBlock is functionally the same as NewStore, however it sets the
// crypto.Block to use on the returned Store. This will allow for encryption of
// keys during creation.
func NewStoreWithBlock(db *sqlx.DB, block *crypto.Block, mm ...database.Model) *Store {
	s := NewStore(db, mm...)
	s.block = block
	return s
}

// FromContext returns the Key model from the given context, if any.
func FromContext(ctx context.Context) (*Key, bool) {
	k, ok := ctx.Value("key").(*Key)
	return k, ok
}

// LoadRelations loads all of the available relations for the given Key models
// using the given loaders available.
func LoadRelations(loaders *database.Loaders, kk ...*Key) error {
	mm := database.ModelSlice(len(kk), Model(kk))
	return errors.Err(database.LoadRelations(relations, loaders, mm...))
}

// Model is called along with database.ModelSlice to convert the given slice of
// Key models to a slice of database.Model interfaces.
func Model(kk []*Key) func(int) database.Model {
	return func(i int) database.Model {
		return kk[i]
	}
}

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either user.User or namespace.Namespace.
func (k *Key) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			k.User = m.(*user.User)
		case *namespace.Namespace:
			k.Namespace = m.(*namespace.Namespace)
		}
	}
}

// SetPrimary implements the database.Model interface.
func (k *Key) SetPrimary(id int64) { k.ID = id }

// Primary implements the database.Model interface.
func (k *Key) Primary() (string, int64) { return "id", k.ID }

func (k *Key) Endpoint(uri ...string) string {
	if len(uri) > 0 {
		return "/keys/" + strconv.FormatInt(k.ID, 10) + "/" + strings.Join(uri, "/")
	}
	return "/keys/" + strconv.FormatInt(k.ID, 10)
}

// IsZero implements the database.Model interface.
func (k *Key) IsZero() bool {
	return k == nil || k.ID == 0 &&
		k.UserID == 0 &&
		!k.NamespaceID.Valid &&
		k.Name == "" &&
		len(k.Key) == 0 &&
		k.Config == "" &&
		k.CreatedAt == time.Time{}
}

// JSON implements the database.Model interface. This will return a map with the
// current Image values under each key. If any of the User, or Namespace bound
// models exist on the Artifact, then the JSON representation of these models
// will be in the returned map, under the user, and namespace keys respectively.
func (k *Key) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":           k.ID,
		"user_id":      k.UserID,
		"namespace_id": nil,
		"name":         k.Name,
		"config":       k.Config,
		"created_at":   k.CreatedAt.Format(time.RFC3339),
		"updated_at":   k.UpdatedAt.Format(time.RFC3339),
		"url":          addr + k.Endpoint(),
	}

	if k.NamespaceID.Valid {
		json["namespace_id"] = k.NamespaceID.Int64
	}

	for name, m := range map[string]database.Model{
		"user":      k.User,
		"namespace": k.Namespace,
	} {
		if !m.IsZero() {
			json[name] = m.JSON(addr)
		}
	}
	return json
}

// Values implements the database.Model interface. This will return a map with
// the following values, user_id, namespace_id, name, key, and config.
func (k *Key) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":      k.UserID,
		"namespace_id": k.NamespaceID,
		"name":         k.Name,
		"key":          k.Key,
		"config":       k.Config,
	}
}

// New returns a new Key binding any non-nil models to it from the current Store.
func (s *Store) New() *Key {
	k := &Key{
		User:      s.User,
		Namespace: s.Namespace,
	}

	if s.User != nil {
		k.UserID = s.User.ID
	}

	if s.Namespace != nil {
		k.NamespaceID = sql.NullInt64{
			Int64: s.Namespace.ID,
			Valid: true,
		}
	}
	return k
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

// Create creates a new key with the given name and config. The given key string
// should be the contents of the key itself, this will be encrypted with the
// underlying crypto.Block that is set on the Store. If no crypto.Block is set
// on the Store then this will error.
func (s *Store) Create(name, key, config string) (*Key, error) {
	if s.block == nil {
		return nil, errors.New("nil block cipher")
	}

	b, err := s.block.Encrypt([]byte(key))

	if err != nil {
		return nil, errors.Err(err)
	}

	k := s.New()
	k.Name = strings.Replace(name, " ", "_", -1)
	k.Key = b
	k.Config = config
	k.CreatedAt = time.Now()

	err = s.Store.Create(table, k)
	return k, errors.Err(err)
}

// Update updates the key with the given id, and set's the new namespace for
// the key, and the new config to use.
func (s *Store) Update(id, namespaceId int64, config string) error {
	q := query.Update(
		query.Table(table),
		query.Set("namespace_id", sql.NullInt64{
			Int64: namespaceId,
			Valid: namespaceId > 0,
		}),
		query.Set("config", config),
		query.SetRaw("updated_at", "NOW()"),
		query.Where("id", "=", id),
	)

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// Delete removes all of the keys from the database with the given list of ids.
func (s *Store) Delete(ids ...int64) error {
	mm := make([]database.Model, 0, len(ids))

	for _, id := range ids {
		mm = append(mm, &Key{ID: id})
	}
	return errors.Err(s.Store.Delete(table, mm...))
}

// Paginate returns the database.Paginator for the keys table for the given page.
func (s *Store) Paginate(page int64, opts ...query.Option) (database.Paginator, error) {
	paginator, err := s.Store.Paginate(table, page, opts...)
	return paginator, errors.Err(err)
}

// All returns a slice of Key models, applying each query.Option that is given.
func (s *Store) All(opts ...query.Option) ([]*Key, error) {
	kk := make([]*Key, 0)

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		database.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.All(&kk, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, k := range kk {
		k.User = s.User
		k.Namespace = s.Namespace
	}
	return kk, errors.Err(err)
}

// Index returns the paginated results from the keys table depending on the
// values that are present in url.Values. Detailed below are the values that
// are used from the given url.Values,
//
// search - This applies the database.Search query.Option using the value of
// name
func (s *Store) Index(vals url.Values, opts ...query.Option) ([]*Key, database.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		database.Search("name", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, opts...)

	if err != nil {
		return []*Key{}, paginator, errors.Err(err)
	}

	kk, err := s.All(append(
		opts,
		query.OrderAsc("key"),
		query.Limit(database.PageLimit),
		query.Offset(paginator.Offset),
	)...)
	return kk, paginator, errors.Err(err)
}

// Get returns a single Key model, applying each query.Option that is given.
func (s *Store) Get(opts ...query.Option) (*Key, error) {
	k := &Key{
		User:      s.User,
		Namespace: s.Namespace,
	}

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		database.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.Get(k, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return k, errors.Err(err)
}

// Load loads in a slice of Key models where the given key is in the list
// of given vals. Each database is loaded individually via a call to the given
// load callback. This method calls Store.All under the hood, so any
// bound models will impact the models being loaded.
func (s *Store) Load(key string, vals []interface{}, load database.LoaderFunc) error {
	kk, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, k := range kk {
			load(i, k)
		}
	}
	return nil
}
