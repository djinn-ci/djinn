package model

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"
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

type KeyStore struct {
	Store

	User      *User
	Namespace *Namespace
}

func keyToInterface(kk []*Key) func(i int) Interface {
	return func(i int) Interface {
		return kk[i]
	}
}

func (k *Key) LoadNamespace() error {
	var err error

	namespaces := NamespaceStore{
		Store: Store{
			DB: k.DB,
		},
	}

	k.Namespace, err = namespaces.Find(k.NamespaceID.Int64)

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

func (s KeyStore) All(opts ...query.Option) ([]*Key, error) {
	kk := make([]*Key, 0)

	opts = append(opts, ForCollaborator(s.User), ForNamespace(s.Namespace))

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

func (s KeyStore) Index(opts ...query.Option) ([]*Key, error) {
	kk, err := s.All(opts...)

	if err != nil {
		return kk, errors.Err(err)
	}

	if err := s.LoadNamespaces(kk); err != nil {
		return kk, errors.Err(err)
	}

	nn := make([]*Namespace, 0, len(kk))

	for _, k := range kk {
		if k.Namespace != nil {
			nn = append(nn, k.Namespace)
		}
	}

	namespaces := NamespaceStore{
		Store: s.Store,
	}

	err = namespaces.LoadUsers(nn)

	return kk, errors.Err(err)
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
		User: s.User,
	}

	if s.User != nil {
		k.UserID = s.User.ID
	}

	return k
}

func (s KeyStore) findBy(col string, val interface{}) (*Key, error) {
	k := &Key{
		Model: Model{
			DB: s.DB,
		},
		User: s.User,
	}

	err := s.FindBy(k, KeyTable, col, val)

	return k, errors.Err(err)
}

func (s KeyStore) Find(id int64) (*Key, error) {
	k, err := s.findBy("id", id)

	return k, errors.Err(err)
}

func (s KeyStore) FindByName(name string) (*Key, error) {
	k, err := s.findBy("name", name)

	return k, errors.Err(err)
}

func (s KeyStore) Update(kk ...*Key) error {
	models := interfaceSlice(len(kk), keyToInterface(kk))

	return errors.Err(s.Store.Update(KeyTable, models...))
}
