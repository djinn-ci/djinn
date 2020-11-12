package build

import (
	"database/sql"

	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/key"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Key is the type that represents an SSH key that has been placed into the
// build environment.
type Key struct {
	ID       int64         `db:"id"`
	BuildID  int64         `db:"build_id"`
	KeyID    sql.NullInt64 `db:"key_id"`
	Name     string        `db:"name"`
	Key      []byte        `db:"key"`
	Config   string        `db:"config"`
	Location string        `db:"location"`

	Build *Build `db:"-"`
}

// KeyStore is the type for creating and modifying Key models in the database.
// The KeyStore type uses an underlying crypto.Block for encrypting the SSH key
// itself when being stored in the database.
type KeyStore struct {
	database.Store

	block *crypto.Block
	Build *Build
	Key   *key.Key
}

var (
	_ database.Model  = (*Key)(nil)
	_ database.Binder = (*KeyStore)(nil)
	_ database.Loader = (*KeyStore)(nil)

	keyTable = "build_keys"
)

// NewKeyStore returns a new KeyStore for querying the build_jobs table. Each
// database passed to this function will be bound to the returned JobStore.
func NewKeyStore(db *sqlx.DB, mm ...database.Model) *KeyStore {
	s := &KeyStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// NewKeyStoreWithblock is functionally the same as NewStore, however it sets
// the given crypto.Block on the newly returned KeyStore.
func NewKeyStoreWithBlock(db *sqlx.DB, block *crypto.Block, mm ...database.Model) *KeyStore {
	s := NewKeyStore(db, mm...)
	s.block = block
	return s
}

// KeyModel is called along with database.ModelSlice to convert the given slice of
// Key models to a slice of database.Model interfaces.
func KeyModel(kk []*Key) func(int) database.Model {
	return func(i int) database.Model {
		return kk[i]
	}
}

// Bind implements the database.Binder interface. This will only bind the
// models if they are pointers to a Build model.
func (k *Key) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *Build:
			k.Build = v
		}
	}
}

// SetPrimary implements the database.Model interface.
func (k *Key) SetPrimary(id int64) { k.ID = id }

// Primary implements the database.Model interface.
func (k *Key) Primary() (string, int64) { return "id", k.ID }

// IsZero implements the database.Model interface.
func (k *Key) IsZero() bool {
	return k == nil || k.ID == 0 &&
		k.BuildID == 0 &&
		!k.KeyID.Valid &&
		k.Name == "" &&
		k.Config == "" &&
		k.Location == ""
}

// JSON implements the database.Model interface. This will return a map with the
// current Key's values under each key.
func (k *Key) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":       k.ID,
		"build_id": k.BuildID,
		"config":   k.Config,
		"location": k.Location,
	}

	if k.KeyID.Valid {
		json["key_id"] = k.KeyID.Int64
	}

	if !k.Build.IsZero() {
		json["build"] = k.Build.JSON(addr)
	}
	return json
}

// Endpoint implements the database.Model interface. This returns an empty
// string.
func (*Key) Endpoint(_ ...string) string { return "" }

// Values implements the database.Model interface. This will return a map with
// the following values, build_id, key_id, name, key, config, and location.
func (k *Key) Values() map[string]interface{} {
	return map[string]interface{}{
		"build_id": k.BuildID,
		"key_id":   k.KeyID,
		"name":     k.Name,
		"key":      k.Key,
		"config":   k.Config,
		"location": k.Location,
	}
}

// New returns a new Key binding any non-nil models to it from the current
// KeyStore.
func (s *KeyStore) New() *Key {
	k := &Key{
		Build: s.Build,
	}

	if s.Build != nil {
		k.BuildID = s.Build.ID
	}

	if s.Key != nil {
		k.KeyID = sql.NullInt64{
			Int64: s.Key.ID,
			Valid: true,
		}
	}
	return k
}

// Bind implements the database.Binder interface. This will only bind the
// models if they are pointers to a Build model.
func (s *KeyStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *Build:
			s.Build = v
		}
	}
}

// Copy copies each given key.Key into a build Key, and returns the slice of
// newly created Key models.
func (s *KeyStore) Copy(kk ...*key.Key) ([]*Key, error) {
	bkk := make([]*Key, 0, len(kk))

	for _, k := range kk {
		bk := s.New()
		bk.KeyID = sql.NullInt64{
			Int64: k.ID,
			Valid: true,
		}
		bk.Key = k.Key
		bk.Config = k.Config
		bk.Location = "/root/.ssh/" + k.Name

		bkk = append(bkk, bk)
	}

	err := s.Store.Create(keyTable, database.ModelSlice(len(bkk), KeyModel(bkk))...)
	return bkk, errors.Err(err)
}

// All returns a slice of Key models, applying each query.Option that is given.
// Each database that is bound to the store will be applied to the list of query
// options via database.Where.
func (s *KeyStore) All(opts ...query.Option) ([]*Key, error) {
	kk := make([]*Key, 0)

	opts = append([]query.Option{
		database.Where(s.Build, "build_id"),
	}, opts...)

	err := s.Store.All(&kk, keyTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, k := range kk {
		k.Build = s.Build
	}
	return kk, errors.Err(err)
}

// Get returns a single Key database, applying each query.Option that is given.
// Each database that is bound to the store will be applied to the list of query
// options via database.Where.
func (s *KeyStore) Get(opts ...query.Option) (*Key, error) {
	k := &Key{
		Build: s.Build,
	}

	opts = append([]query.Option{
		database.Where(s.Build, "build_id"),
	}, opts...)

	err := s.Store.Get(k, keyTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return k, errors.Err(err)
}

// Load loads in a slice of Key models where the given key is in the list of
// given vals. Each database is loaded individually via a call to the given load
// callback. This method calls KeyStore.All under the hood, so any bound models
// will impact the models being loaded.
func (s *KeyStore) Load(key string, vals []interface{}, load database.LoaderFunc) error {
	kk, err := s.All(query.Where(key, "IN", query.List(vals...)))

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

func (s *KeyStore) getKeyToPlace(name string) (*Key, error) {
	k, err := s.Get(query.Where("name", "=", query.Arg(name)))

	if err != nil {
		return nil, errors.Err(err)
	}

	if k.IsZero() {
		return nil, errors.New("cannot find key: " + name)
	}
	return k, nil
}
