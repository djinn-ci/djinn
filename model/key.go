package model

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/andrewpillar/thrall/errors"

	"github.com/andrewpillar/query"
)

type Key struct {
	Model

	UserID      int64         `db:"user_id"`
	NamespaceID sql.NullInt64 `db:"namespace_id"`
	Name        string        `db:"name"`
	Key         []byte        `db:"key"`
	Config      string        `db:"config"`

	User      *User
	Namespace *Namespace
}

type BuildKey struct {
	Model

	BuildID  int64         `db:"build_id"`
	KeyID    sql.NullInt64 `db:"key_id"`
	Name     string        `db:"name"`
	Key      []byte        `db:"key"`
	Config   string        `db:"config"`
	Location string        `db:"location"`

	Build *Build `db:"-"`
}

type KeyStore struct {
	Store

	User      *User
	Namespace *Namespace
}

type BuildKeyStore struct {
	Store

	Build *Build
}

func buildKeyToInterface(bkk []*BuildKey) func(i int) Interface {
	return func(i int) Interface {
		return bkk[i]
	}
}

func keyToInterface(kk []*Key) func(i int) Interface {
	return func(i int) Interface {
		return kk[i]
	}
}

func (bk BuildKey) Values() map[string]interface{} {
	return map[string]interface{}{
		"build_id":   bk.BuildID,
		"key_id":     bk.KeyID,
		"name":       bk.Name,
		"key":        bk.Key,
		"config":     bk.Config,
		"location":   bk.Location,
		"created_at": bk.CreatedAt,
		"updated_at": bk.UpdatedAt,
	}
}

func (k *Key) LoadNamespace() error {
	var err error

	namespaces := NamespaceStore{
		Store: Store{
			DB: k.DB,
		},
	}

	k.Namespace, err = namespaces.Get(query.Where("id", "=", k.NamespaceID.Int64))

	return errors.Err(err)
}

func (k Key) UIEndpoint(uri ...string) string {
	endpoint := fmt.Sprintf("/keys/%v", k.ID)

	if len(uri) > 0 {
		endpoint = fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}

	return endpoint
}

func (k Key) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":      k.UserID,
		"namespace_id": k.NamespaceID,
		"name":         k.Name,
		"key":          k.Key,
		"config":       k.Config,
	}
}

func (s BuildKeyStore) All(opts ...query.Option) ([]*BuildKey, error) {
	bkk := make([]*BuildKey, 0)

	opts = append(opts, ForBuild(s.Build))

	err := s.Store.All(&bkk, BuildKeyTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, bk := range bkk {
		bk.DB = s.DB
		bk.Build = s.Build
	}

	return bkk, errors.Err(err)
}

func (s BuildKeyStore) Create(bkk ...*BuildKey) error {
	models := interfaceSlice(len(bkk), buildKeyToInterface(bkk))

	return errors.Err(s.Store.Create(BuildKeyTable, models...))
}

func (s BuildKeyStore) Copy(kk []*Key) error {
	if len(kk) == 0 {
		return nil
	}

	bkk := make([]*BuildKey, 0, len(kk))

	for _, k := range kk {
		bk := s.New()
		bk.KeyID = sql.NullInt64{
			Int64: k.ID,
			Valid: true,
		}
		bk.Name = k.Name
		bk.Key = k.Key
		bk.Config = k.Config
		bk.Location = "/root/.ssh/" + bk.Name

		bkk = append(bkk, bk)
	}

	return errors.Err(s.Create(bkk...))
}

func (s BuildKeyStore) New() *BuildKey {
	bk := &BuildKey{
		Model: Model{
			DB: s.DB,
		},
		Build: s.Build,
	}

	if s.Build != nil {
		bk.BuildID = s.Build.ID
	}

	return bk
}

func (s KeyStore) All(opts ...query.Option) ([]*Key, error) {
	kk := make([]*Key, 0)

	opts = append([]query.Option{ForCollaborator(s.User), ForNamespace(s.Namespace)}, opts...)

	err := s.Store.All(&kk, KeyTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, k := range kk {
		k.DB = s.DB

		if s.User != nil {
			k.User = s.User
		}
	}

	return kk, errors.Err(err)
}

func (s KeyStore) Create(kk ...*Key) error {
	models := interfaceSlice(len(kk), keyToInterface(kk))

	return errors.Err(s.Store.Create(KeyTable, models...))
}

func (s KeyStore) Delete(kk ...*Key) error {
	models := interfaceSlice(len(kk), keyToInterface(kk))

	return errors.Err(s.Store.Delete(KeyTable, models...))
}

func (s KeyStore) Get(opts ...query.Option) (*Key, error) {
	k := &Key{
		Model: Model{
			DB: s.DB,
		},
		User:      s.User,
		Namespace: s.Namespace,
	}

	baseOpts := []query.Option{
		query.Columns("*"),
		query.From(KeyTable),
		ForUser(s.User),
		ForNamespace(s.Namespace),
	}

	q := query.Select(append(baseOpts, opts...)...)

	err := s.Store.Get(k, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return k, errors.Err(err)
}

func (s KeyStore) loadNamespace(kk []*Key) func(i int, n *Namespace) {
	return func(i int, n *Namespace) {
		k := kk[i]

		if k.NamespaceID.Int64 == n.ID {
			k.Namespace = n
		}
	}
}

func (s KeyStore) LoadNamespaces(kk []*Key) error {
	if len(kk) == 0 {
		return nil
	}

	models := interfaceSlice(len(kk), keyToInterface(kk))

	namespaces := NamespaceStore{
		Store: s.Store,
	}

	err := namespaces.Load(mapKey("namespace_id", models), s.loadNamespace(kk))

	return errors.Err(err)
}

func (s KeyStore) New() *Key {
	k := &Key{
		Model: Model{
			DB: s.DB,
		},
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

func (s KeyStore) Paginate(page int64, opts ...query.Option) (Paginator, error) {
	paginator, err := s.Store.Paginate(KeyTable, page, opts...)

	return paginator, errors.Err(err)
}

func (s KeyStore) Update(kk ...*Key) error {
	models := interfaceSlice(len(kk), keyToInterface(kk))

	return errors.Err(s.Store.Update(KeyTable, models...))
}
