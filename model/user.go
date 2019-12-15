package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"

	"github.com/andrewpillar/query"

	"github.com/lib/pq"
)

type User struct {
	Model

	Email     string      `db:"email"`
	Username  string      `db:"username"`
	Password  []byte      `db:"password"`
	DeletedAt pq.NullTime `db:"deleted_at"`
	Connected bool        `db:"-"`
}

type UserStore struct {
	Store
}

func userToInterface(uu []*User) func(i int) Interface {
	return func(i int) Interface {
		return uu[i]
	}
}

func (s UserStore) All(opts ...query.Option) ([]*User, error) {
	uu := make([]*User, 0)

	err := s.Store.All(&uu, UserTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, u := range uu {
		u.DB = s.DB
	}

	return uu, nil
}

func (s UserStore) Create(uu ...*User) error {
	models := interfaceSlice(len(uu), userToInterface(uu))

	return errors.Err(s.Store.Create(UserTable, models...))
}

func (s UserStore) Get(opts ...query.Option) (*User, error) {
	u := &User{
		Model: Model{
			DB: s.DB,
		},
	}

	baseOpts := []query.Option{
		query.Columns("*"),
		query.From(UserTable),
	}

	q := query.Select(append(baseOpts, opts...)...)

	err := s.Store.Get(u, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return u, errors.Err(err)
}

func (s UserStore) Load(ids []interface{}, load func(i int, u *User)) error {
	if len(ids) == 0 {
		return nil
	}

	uu, err := s.All(query.Where("id", "IN", ids...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range ids {
		for _, u := range uu {
			load(i, u)
		}
	}

	return nil
}

func (s UserStore) New() *User {
	u := &User{
		Model: Model{
			DB: s.DB,
		},
	}

	return u
}

func (s UserStore) Update(uu ...*User) error {
	models := interfaceSlice(len(uu), userToInterface(uu))

	return errors.Err(s.Store.Update(UserTable, models...))
}

func (u *User) BuildStore() BuildStore {
	return BuildStore{
		Store: Store{
			DB: u.DB,
		},
		User: u,
	}
}

func (u *User) CollaboratorStore() CollaboratorStore {
	return CollaboratorStore{
		Store: Store{
			DB: u.DB,
		},
		User: u,
	}
}

func (u *User) ImageStore() ImageStore {
	return ImageStore{
		Store: Store{
			DB: u.DB,
		},
		User: u,
	}
}

func (u *User) InviteStore() InviteStore {
	return InviteStore{
		Store: Store{
			DB: u.DB,
		},
	}
}

func (u *User) IsZero() bool {
	return u.Model.IsZero() &&
		u.Email == "" &&
		u.Username == "" &&
		len(u.Password) == 0 &&
		!u.DeletedAt.Valid
}

func (u *User) KeyStore() KeyStore {
	return KeyStore{
		Store: Store{
			DB: u.DB,
		},
		User: u,
	}
}

func (u *User) NamespaceStore() NamespaceStore {
	return NamespaceStore{
		Store: Store{
			DB: u.DB,
		},
		User: u,
	}
}

func (u *User) ObjectStore() ObjectStore {
	return ObjectStore{
		Store: Store{
			DB: u.DB,
		},
		User: u,
	}
}

func (u *User) ProviderStore() ProviderStore {
	return ProviderStore{
		Store: Store{
			DB: u.DB,
		},
		User: u,
	}
}

func (u *User) RepoStore() RepoStore {
	return RepoStore{
		Store: Store{
			DB: u.DB,
		},
		User: u,
	}
}

func (u User) Values() map[string]interface{} {
	return map[string]interface{}{
		"email":    u.Email,
		"username": u.Username,
		"password": u.Password,
	}
}

func (u *User) VariableStore() VariableStore {
	return VariableStore{
		Store: Store{
			DB: u.DB,
		},
		User: u,
	}
}
