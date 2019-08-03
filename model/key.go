package model

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"
)

type Key struct {
	Model

	UserID  int64  `db:"user_id"`
	Name    string `db:"name"`
	Key     []byte `db:"key"`
	Config  string `db:"config"`

	User *User
}

type KeyStore struct {
	*sqlx.DB

	User *User
}

func (k Key) AccessibleBy(u *User) bool {
	if u == nil {
		return false
	}

	return k.UserID == u.ID
}

func (k *Key) Create() error {
	q := Insert(
		Table("keys"),
		Columns("user_id", "name", "key", "config"),
		Values(k.UserID, k.Name, k.Key, k.Config),
		Returning("id", "created_at", "updated_at"),
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
	q := Delete(
		Table("keys"),
		WhereEq("id", k.ID),
	)

	stmt, err := k.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(q.Args()...)

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
	q := Update(
		Table("keys"),
		Set("config", k.Config),
		WhereEq("id", k.ID),
		Returning("updated_at"),
	)

	stmt, err := k.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&k.UpdatedAt))
}

func (ks KeyStore) All(opts ...Option) ([]*Key, error) {
	kk := make([]*Key, 0)

	opts = append([]Option{Columns("*")}, opts...)

	q := Select(append(opts, ForUser(ks.User), Table("keys"))...)

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

	q := Select(
		Columns("*"),
		Table("keys"),
		WhereEq(col, val),
		ForUser(ks.User),
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
