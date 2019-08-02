package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type User struct {
	model

	Email     string      `db:"email"`
	Username  string      `db:"username"`
	Password  []byte      `db:"password"`
	DeletedAt pq.NullTime `db:"deleted_at"`
}

type UserStore struct {
	*sqlx.DB
}

func (us UserStore) New() *User {
	u := &User{
		model: model{
			DB: us.DB,
		},
	}

	return u
}

func (u *User) BuildStore() BuildStore {
	return BuildStore{
		DB:   u.DB,
		User: u,
	}
}

func (u *User) KeyStore() KeyStore {
	return KeyStore{
		DB:   u.DB,
		User: u,
	}
}

func (u *User) NamespaceStore() NamespaceStore {
	return NamespaceStore{
		DB:   u.DB,
		User: u,
	}
}

func (u *User) ObjectStore() ObjectStore {
	return ObjectStore{
		DB:   u.DB,
		User: u,
	}
}

func (u *User) VariableStore() VariableStore {
	return VariableStore{
		DB:   u.DB,
		User: u,
	}
}

func (u *User) Create() error {
	q := Insert(
		Table("users"),
		Columns("email", "username", "password"),
		Values(u.Email, u.Username, u.Password),
		Returning("id", "created_at", "updated_at"),
	)

	stmt, err := u.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt))
}

func (u *User) Destroy() error {
	q := Update(
		Table("users"),
		SetRaw("deleted_at", "NOW()"),
		WhereEq("id", u.ID),
		Returning("deleted_at"),
	)

	stmt, err := u.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&u.DeletedAt))
}

func (u *User) IsZero() bool {
	return u.model.IsZero() &&
		u.Email == "" &&
		u.Username == "" &&
		len(u.Password) == 0 &&
		!u.DeletedAt.Valid
}

func (u *User) Update() error {
	q := Update(
		Table("users"),
		Set("email", u.Email),
		Set("password", u.Password),
		SetRaw("updated_at", "NOW()"),
		WhereEq("id", u.ID),
		Returning("updated_at"),
	)

	stmt, err := u.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&u.UpdatedAt))
}

func (us UserStore) All(opts ...Option) ([]*User, error) {
	uu := make([]*User, 0)

	opts = append([]Option{Columns("*")}, opts...)

	q := Select(append(opts, Table("users"))...)

	err := us.Select(&uu, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, u := range uu {
		u.DB = us.DB
	}

	return uu, nil
}

func (us UserStore) Find(id int64) (*User, error) {
	u := &User{
		model: model{
			DB: us.DB,
		},
	}

	q := Select(Columns("*"), Table("users"), WhereEq("id", id))

	err := us.Get(u, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return u, errors.Err(err)
}

func (us UserStore) FindByEmail(email string) (*User, error) {
	u := &User{
		model: model{
			DB: us.DB,
		},
	}

	q := Select(Columns("*"), Table("users"), WhereEq("email", u.Email))

	err := us.Get(u, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return u, errors.Err(err)
}

func (us UserStore) FindByHandle(handle string) (*User, error) {
	u := &User{
		model: model{
			DB: us.DB,
		},
	}

	q := Select(
		Columns("*"),
		Table("users"),
		Or(
			WhereEq("username", handle),
			WhereEq("email", handle),
		),
	)

	err := us.Get(u, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return u, errors.Err(err)
}

func (us UserStore) FindByUsername(username string) (*User, error) {
	u := &User{
		model: model{
			DB: us.DB,
		},
	}

	q := Select(Columns("*"), Table("users"), WhereEq("username", username))

	err := us.Get(u, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return u, errors.Err(err)
}
