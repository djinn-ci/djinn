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

func NewKeyStore(db *sqlx.DB, mm ...model.Model) KeyStore {
	s := KeyStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func KeyModel(kk []*Key) func(int) model.Model {
	return func(i int) model.Model {
		return kk[i]
	}
}

func (i *keyInfo) Name() string { return i.name }
func (i *keyInfo) Size() int64 { return i.size }
func (*keyInfo) Mode() os.FileMode { return os.FileMode(0600) }
func (i *keyInfo) ModTime() time.Time { return i.modTime }
func (i *keyInfo) IsDir() bool { return false }
func (i *keyInfo) Sys() interface{} { return nil }

func (k *Key) Bind(mm ...model.Model) {
	if k == nil {
		return
	}

	for _, m := range mm {
		switch m.(type) {
		case *Build:
			k.Build = m.(*Build)
		}
	}
}

func (*Key) Kind() string { return "build_key" }

func (k *Key) SetPrimary(id int64) {
	if k == nil {
		return
	}
	k.ID = id
}

func (k *Key) Primary() (string, int64) {
	if k == nil {
		return "id", 0
	}
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

func (*Key) Endpoint(_ ...string) string { return "" }

func (k *Key) Values() map[string]interface{} {
	if k == nil {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"build_id": k.BuildID,
		"key_id":   k.KeyID,
		"name":     k.Name,
		"config":   k.Config,
		"location": k.Location,
	}
}

func (s KeyStore) New() *Key {
	k := &Key{
		Build: s.Build,
	}

	if s.Build != nil {
		k.Build = s.Build
	}
	return k
}

func (s *KeyStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			s.Build = m.(*Build)
		}
	}
}

func (s KeyStore) Create(kk ...*Key) error {
	models := model.Slice(len(kk), KeyModel(kk))
	return errors.Err(s.Store.Create(keyTable, models...))
}

func (s KeyStore) All(opts ...query.Option) ([]*Key, error) {
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

func (s KeyStore) Get(opts ...query.Option) (*Key, error) {
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

func (s KeyStore) Load(key string, vals []interface{}, load model.LoaderFunc) error {
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

func (s KeyStore) getKeyToPlace(name string) (*Key, error) {
	k, err := s.Get(query.Where("name", "=", name))

	if err != nil {
		return nil, errors.Err(err)
	}

	if k.IsZero() {
		return nil, errors.New("cannot find key: "+name)
	}
	return k, nil
}

func (s KeyStore) Place(name string, w io.Writer) (int64, error) {
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

func (s KeyStore) Stat(name string) (os.FileInfo, error) {
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
