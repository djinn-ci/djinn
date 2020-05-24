package build

import (
	"database/sql"
	"io"
	"os"
	"time"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/runner"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

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

// keyInfo is a bare minimum implementation of the os.FileInfo interface just
// so we can return it from the KeyStore.Stat call.
type keyInfo struct {
	name    string
	size    int64
	modTime time.Time
}

type KeyStore struct {
	model.Store

	Build *Build
}

var (
	_ os.FileInfo   = (*keyInfo)(nil)
	_ model.Model   = (*Key)(nil)
	_ model.Binder  = (*KeyStore)(nil)
	_ model.Loader  = (*KeyStore)(nil)
	_ runner.Placer = (*KeyStore)(nil)

	keyTable = "build_keys"
)

// NewKeyStore returns a new KeyStore for querying the build_jobs table. Each
// model passed to this function will be bound to the returned JobStore.
func NewKeyStore(db *sqlx.DB, mm ...model.Model) *KeyStore {
	s := &KeyStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// KeyModel is called along with model.Slice to convert the given slice of
// Key models to a slice of model.Model interfaces.
func KeyModel(kk []*Key) func(int) model.Model {
	return func(i int) model.Model {
		return kk[i]
	}
}

// keyInfo method stubs for os.FileInfo interface.
func (i *keyInfo) Name() string { return i.name }
func (i *keyInfo) Size() int64 { return i.size }
func (*keyInfo) Mode() os.FileMode { return os.FileMode(0600) }
func (i *keyInfo) ModTime() time.Time { return i.modTime }
func (i *keyInfo) IsDir() bool { return false }
func (i *keyInfo) Sys() interface{} { return nil }

// Bind the given models to the current Key. This will only bind the model if
// they are one of the following,
//
// - *Build
func (k *Key) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			k.Build = m.(*Build)
		}
	}
}

func (k *Key) SetPrimary(id int64) {
	k.ID = id
}

func (k *Key) Primary() (string, int64) {
	return "id", k.ID
}

func (k *Key) IsZero() bool {
	return k == nil || k.ID == 0 &&
		k.BuildID == 0 &&
		!k.KeyID.Valid &&
		k.Name == "" &&
		k.Config == "" &&
		k.Location == ""
}

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

func (*Key) Endpoint(_ ...string) string { return "" }

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
	return k
}

// Bind the given models to the current KeyStore. This will only bind the model
// if they are one of the following,
//
// - *Build
func (s *KeyStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			s.Build = m.(*Build)
		}
	}
}

// Create inserts the given Key models into the build_keys table.
func (s *KeyStore) Create(kk ...*Key) error {
	models := model.Slice(len(kk), KeyModel(kk))
	return errors.Err(s.Store.Create(keyTable, models...))
}

// All returns a slice of Key models, applying each query.Option that is given.
// Each model that is bound to the store will be applied to the list of query
// options via model.Where.
func (s *KeyStore) All(opts ...query.Option) ([]*Key, error) {
	kk := make([]*Key, 0)

	opts = append([]query.Option{
		model.Where(s.Build, "build_id"),
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

// Get returns a single Key model, applying each query.Option that is given.
// Each model that is bound to the store will be applied to the list of query
// options via model.Where.
func (s *KeyStore) Get(opts ...query.Option) (*Key, error) {
	k := &Key{
		Build: s.Build,
	}

	opts = append([]query.Option{
		model.Where(s.Build, "build_id"),
	}, opts...)

	err := s.Store.Get(k, keyTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return k, errors.Err(err)
}

// Load loads in a slice of Key models where the given key is in the list of
// given vals. Each model is loaded individually via a call to the given load
// callback. This method calls KeyStore.All under the hood, so any bound models
// will impact the models being loaded.
func (s *KeyStore) Load(key string, vals []interface{}, load model.LoaderFunc) error {
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

func (s *KeyStore) getKeyToPlace(name string) (*Key, error) {
	k, err := s.Get(query.Where("name", "=", name))

	if err != nil {
		return nil, errors.Err(err)
	}

	if k.IsZero() {
		return nil, errors.New("cannot find key: "+name)
	}
	return k, nil
}

// Place looks up the Key by the given name, and decrypts its content so it can
// be written to the given io.Writer.
func (s *KeyStore) Place(name string, w io.Writer) (int64, error) {
	k, err := s.getKeyToPlace(name)

	if err != nil {
		return 0, errors.Err(err)
	}

	b, err := crypto.Decrypt(k.Key)

	if err != nil {
		return 0, errors.Err(err)
	}

	n, err := w.Write(b)
	return int64(n), errors.Err(err)
}

// Stat returns an implementation of os.FileInfo for the given SSH key. The
// ModTime is always going to be when this method was invoked, and the length
// will be the length of the decrypted SSH key itself.
func (s *KeyStore) Stat(name string) (os.FileInfo, error) {
	k, err := s.getKeyToPlace(name)

	if err != nil {
		return nil, errors.Err(err)
	}

	b, err := crypto.Decrypt(k.Key)

	return &keyInfo{
		name:    k.Name,
		size:    int64(len(b)),
		modTime: time.Now(),
	}, errors.Err(err)
}
