package model

import (
	"database/sql"
	"time"

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

func (u *User) BuildList(status string) ([]*Build, error) {
	var (
		bb  []*Build
		err error
	)

	builds := u.BuildStore()

	if status != "" {
		bb, err = builds.ByStatus(status)
	} else {
		bb, err = builds.All()
	}

	if err != nil {
		return bb, errors.Err(err)
	}

	if err != nil {
		return bb, errors.Err(err)
	}

	if err := builds.LoadNamespaces(bb); err != nil {
		return bb, errors.Err(err)
	}

	if err := builds.LoadTags(bb); err != nil {
		return bb, errors.Err(err)
	}

	if err := builds.LoadUsers(bb); err != nil {
		return bb, errors.Err(err)
	}

	nn := make([]*Namespace, 0, len(bb))

	for _, b := range bb {
		if b.Namespace != nil {
			nn = append(nn, b.Namespace)
		}
	}

	ns := NamespaceStore{
		DB: u.DB,
	}

	err = ns.LoadUsers(nn)

	return bb, errors.Err(err)
}

func (u *User) BuildShow(id int64) (*Build, error) {
	b, err := u.BuildStore().Find(id)

	if err != nil {
		return b, errors.Err(err)
	}

	b.User = u

	if err := b.LoadNamespace(); err != nil {
		return b, errors.Err(err)
	}

	if err := b.Namespace.LoadUser(); err != nil {
		return b, errors.Err(err)
	}

	if err := b.LoadTrigger(); err != nil {
		return b, errors.Err(err)
	}

	if err := b.LoadTags(); err != nil {
		return b, errors.Err(err)
	}

	if err := b.LoadStages(); err != nil {
		return b, errors.Err(err)
	}

	err = b.StageStore().LoadJobs(b.Stages)

	return b, errors.Err(err)
}

func (u *User) BuildStore() BuildStore {
	return BuildStore{
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
	stmt, err := u.Prepare(`
		INSERT INTO users (email, username, password)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(u.Email, u.Username, u.Password)

	return errors.Err(row.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt))
}

func (u *User) Destroy() error {
	u.DeletedAt = pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	stmt, err := u.Prepare("UPDATE users SET deleted_at = $1 WHERE id = $2")

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(u.DeletedAt)

	return errors.Err(err)
}

func (u *User) IsZero() bool {
	return u.model.IsZero() &&
           u.Email == "" &&
           u.Username == "" &&
           len(u.Password) == 0 &&
           !u.DeletedAt.Valid
}

func (u *User) Update() error {
	stmt, err := u.Prepare(`
		UPDATE users
		SET email = $1, username = $2, password = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(u.Email, u.Username, u.Password, u.ID)

	return errors.Err(row.Scan(&u.UpdatedAt))
}

func (us UserStore) Find(id int64) (*User, error) {
	u := &User{
		model: model{
			DB: us.DB,
		},
	}

	err := us.Get(u, "SELECT * FROM users WHERE id = $1", id)

	if err == sql.ErrNoRows {
		err = nil

		u.CreatedAt = nil
		u.UpdatedAt = nil
	}

	return u, errors.Err(err)
}

func (us UserStore) FindByEmail(email string) (*User, error) {
	u := &User{
		model: model{
			DB: us.DB,
		},
	}

	err := us.Get(u, "SELECT * FROM users WHERE email = $1", email)

	if err == sql.ErrNoRows {
		err = nil

		u.CreatedAt = nil
		u.UpdatedAt = nil
	}

	return u, errors.Err(err)
}

func (us UserStore) FindByHandle(handle string) (*User, error) {
	u := &User{
		model: model{
			DB: us.DB,
		},
	}

	err := us.Get(u, "SELECT * FROM users WHERE username = $1 OR email = $2", handle, handle)

	if err == sql.ErrNoRows {
		err = nil

		u.CreatedAt = nil
		u.UpdatedAt = nil
	}

	return u, errors.Err(err)
}

func (us UserStore) FindByUsername(username string) (*User, error) {
	u := &User{
		model: model{
			DB: us.DB,
		},
	}

	err := us.Get(u, "SELECT * FROM users WHERE username = $1", username)

	if err == sql.ErrNoRows {
		err = nil

		u.CreatedAt = nil
		u.UpdatedAt = nil
	}

	return u, errors.Err(err)
}

func (us UserStore) In(ids ...int64) ([]*User, error) {
	uu := make([]*User, 0)

	if len(ids) == 0 {
		return uu, nil
	}

	query, args, err := sqlx.In("SELECT * FROM users WHERE id IN (?)", ids)

	if err != nil {
		return uu, errors.Err(err)
	}

	err = us.Select(&uu, us.Rebind(query), args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, u := range uu {
		u.DB = us.DB
	}

	return uu, errors.Err(err)
}
