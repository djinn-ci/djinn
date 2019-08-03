package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type User struct {
	Model

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
		Model: Model{
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
	q := query.Insert(
		query.Table("users"),
		query.Columns("email", "username", "password"),
		query.Values(u.Email, u.Username, u.Password),
		query.Returning("id", "created_at", "updated_at"),
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
	q := query.Update(
		query.Table("users"),
		query.SetRaw("deleted_at", "NOW()"),
		query.WhereEq("id", u.ID),
		query.Returning("deleted_at"),
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
	return u.Model.IsZero() &&
		u.Email == "" &&
		u.Username == "" &&
		len(u.Password) == 0 &&
		!u.DeletedAt.Valid
}

func (u *User) Update() error {
	q := query.Update(
		query.Table("users"),
		query.Set("email", u.Email),
		query.Set("password", u.Password),
		query.SetRaw("updated_at", "NOW()"),
		query.WhereEq("id", u.ID),
		query.Returning("updated_at"),
	)

	stmt, err := u.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&u.UpdatedAt))
}

func (us UserStore) All(opts ...query.Option) ([]*User, error) {
	uu := make([]*User, 0)

	opts = append([]query.Option{query.Columns("*")}, opts...)
	opts = append(opts, query.Table("users"))

	q := query.Select(opts...)

	err := us.Select(&uu, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, u := range uu {
		u.DB = us.DB
	}

	return uu, nil
}

func (us UserStore) findBy(col string, val interface{}) (*User, error) {
	u := &User{
		Model: Model{
			DB: us.DB,
		},
	}

	q := query.Select(query.Columns("*"), query.Table("users"), query.WhereEq(col, val))

	err := us.Get(u, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return u, errors.Err(err)
}

func (us UserStore) Find(id int64) (*User, error) {
	u, err := us.findBy("id", id)

	return u, errors.Err(err)
}

func (us UserStore) FindByEmail(email string) (*User, error) {
	u, err := us.findBy("email", email)

	return u, errors.Err(err)
}

func (us UserStore) FindByHandle(handle string) (*User, error) {
	u := &User{
		Model: Model{
			DB: us.DB,
		},
	}

	q := query.Select(
		query.Columns("*"),
		query.Table("users"),
		query.Or(
			query.WhereEq("username", handle),
			query.WhereEq("email", handle),
		),
	)

	err := us.Get(u, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return u, errors.Err(err)
}

func (us UserStore) FindByUsername(username string) (*User, error) {
	u, err := us.findBy("username", username)

	return u, errors.Err(err)
}
