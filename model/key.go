package model

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"

	"github.com/jmoiron/sqlx"
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
	*sqlx.DB

	User      *User
	Namespace *Namespace
}

func (k Key) AccessibleBy(u *User, a Action) bool {
	if u == nil {
		return false
	}

	return k.UserID == u.ID
}

func (k *Key) Create() error {
	q := query.Insert(
		query.Table("keys"),
		query.Columns("user_id", "namespace_id", "name", "key", "config"),
		query.Values(k.UserID, k.NamespaceID, k.Name, k.Key, k.Config),
		query.Returning("id", "created_at", "updated_at"),
	)

	stmt, err := k.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&k.ID, &k.CreatedAt, &k.UpdatedAt))
}

func (k *Key) Destroy() error {
	q := query.Delete(
		query.Table("keys"),
		query.WhereEq("id", k.ID),
	)

	stmt, err := k.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(q.Args()...)

	return errors.Err(err)
}

func (k *Key) LoadNamespace() error {
	var err error

	namespaces := NamespaceStore{
		DB: k.DB,
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

func (k *Key) Update() error {
	q := query.Update(
		query.Table("keys"),
		query.Set("namespace_id", k.NamespaceID),
		query.Set("config", k.Config),
		query.WhereEq("id", k.ID),
		query.Returning("updated_at"),
	)

	stmt, err := k.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&k.UpdatedAt))
}

func (ks KeyStore) All(opts ...query.Option) ([]*Key, error) {
	kk := make([]*Key, 0)

	opts = append([]query.Option{query.Columns("*")}, opts...)
	opts = append(opts, ForUser(ks.User), ForNamespace(ks.Namespace), query.Table("keys"))

	q := query.Select(opts...)

	err := ks.Select(&kk, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, k := range kk {
		k.DB = ks.DB

		if ks.User != nil {
			k.User = ks.User
		}
	}

	return kk, errors.Err(err)
}

func (ks KeyStore) Index(opts ...query.Option) ([]*Key, error) {
	kk, err := ks.All(opts...)

	if err != nil {
		return kk, errors.Err(err)
	}

	namespaces := NamespaceStore{
		DB: ks.DB,
	}

	ids := make([]interface{}, len(kk), len(kk))

	for i, k := range kk {
		if k.NamespaceID.Valid {
			ids[i] = k.NamespaceID.Int64
		}
	}

	nn := make([]*Namespace, 0, len(ids))
	userIds := make([]interface{}, 0, len(ids))

	err = namespaces.Load(ids, func(i int, n *Namespace) {
		k := kk[i]

		if k.NamespaceID.Int64 == n.ID {
			nn = append(nn, n)
			userIds = append(userIds, n.UserID)

			k.Namespace = n
		}
	})

	if err != nil {
		return kk, errors.Err(err)
	}

	users := UserStore{
		DB: ks.DB,
	}

	err = users.Load(userIds, func(i int, u *User) {
		n := nn[i]

		if n.UserID == u.ID {
			n.User = u
		}
	})

	return kk, errors.Err(err)
}

func (ks KeyStore) New() *Key {
	k := &Key{
		Model: Model{
			DB: ks.DB,
		},
		User: ks.User,
	}

	if ks.User != nil {
		k.UserID = ks.User.ID
	}

	return k
}

func (ks KeyStore) findBy(col string, val interface{}) (*Key, error) {
	k := &Key{
		Model: Model{
			DB: ks.DB,
		},
		User: ks.User,
	}

	q := query.Select(
		query.Columns("*"),
		query.Table("keys"),
		query.WhereEq(col, val),
		ForUser(ks.User),
		ForNamespace(ks.Namespace),
	)

	err := ks.Get(k, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return k, errors.Err(err)
}

func (ks KeyStore) Find(id int64) (*Key, error) {
	k, err := ks.findBy("id", id)

	return k, errors.Err(err)
}

func (ks KeyStore) FindByName(name string) (*Key, error) {
	k, err := ks.findBy("name", name)

	return k, errors.Err(err)
}
