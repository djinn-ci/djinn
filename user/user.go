package user

import (
	"database/sql"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int64       `db:"id"`
	Email     string      `db:"email"`
	Username  string      `db:"username"`
	Password  []byte      `db:"password"`
	CreatedAt time.Time   `db:"created_at"`
	UpdatedAt time.Time   `db:"updated_at"`
	DeletedAt pq.NullTime `db:"deleted_at"`

	Connected   bool                `db:"-"`
	Permissions map[string]struct{} `db:"-"`
}

type Store struct {
	model.Store
}

var (
	_ model.Model  = (*User)(nil)
	_ model.Binder = (*Store)(nil)
	_ model.Loader = (*Store)(nil)

	table             = "users"
	collaboratorTable = "collaborators"

	ErrAuth = errors.New("invalid credentials")
)

func NewStore(db *sqlx.DB, mm ...model.Model) Store {
	s := Store{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func Select(col string, opts ...query.Option) query.Query {
	return query.Select(append([]query.Option{
		query.Columns(col),
		query.From(table),
	}, opts...)...)
}

func WhereHandle(handle string) query.Option {
	return query.Options(
		query.Where("email", "=", handle),
		query.OrWhere("username", "=", handle),
	)
}

func Model(uu []*User) func(int) model.Model {
	return func(i int) model.Model {
		return uu[i]
	}
}

func (u *User) Bind(_ ...model.Model) {}
func (u *User) Kind() string { return "user" }

func (u *User) SetPrimary(id int64) {
	if u == nil {
		return
	}
	u.ID = id
}

func (u *User) Primary() (string, int64) {
	if u == nil {
		return "id", 0
	}
	return "id", u.ID
}

func (u *User) IsZero() bool {
	return u == nil || u.ID == 0 &&
		u.Email == "" &&
		u.Username == "" &&
		len(u.Password) == 0 &&
		u.CreatedAt == time.Time{} &&
		!u.DeletedAt.Valid
}

func (u User) Endpoint(...string) string { return "" }

func (u User) Values() map[string]interface{} {
	return map[string]interface{}{
		"email":      u.Email,
		"username":   u.Username,
		"password":   u.Password,
		"updated_at": u.UpdatedAt,
		"deleted_at": u.DeletedAt,
	}
}

func (s *Store) Bind(_ ...model.Model) {}

func (s Store) All(opts ...query.Option) ([]*User, error) {
	uu := make([]*User, 0)

	err := s.Store.All(&uu, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return uu, errors.Err(err)
}

func (s Store) Load(key string, vals []interface{}, load model.LoaderFunc) error {
	uu, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, u := range uu {
			load(i, u)
		}
	}
	return nil
}

func (s Store) New() *User {
	return &User{}
}

func (s Store) Create(uu ...*User) error {
	models := model.Slice(len(uu), Model(uu))
	return errors.Err(s.Store.Create(table, models...))
}

func (s Store) Update(uu ...*User) error {
	models := model.Slice(len(uu), Model(uu))
	return errors.Err(s.Store.Update(table, models...))
}

func (s Store) Get(opts ...query.Option) (*User, error) {
	u := &User{
		Permissions: map[string]struct{}{
			"build:write": {},
		},
	}

	err := s.Store.Get(u, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return u, errors.Err(err)
}

func (s Store) Auth(handle, password string) (*User, error) {
	u, err := s.Get(WhereHandle(handle))

	if err != nil {
		return u, errors.Err(err)
	}

	if err := bcrypt.CompareHashAndPassword(u.Password, []byte(password)); err != nil {
		return u, ErrAuth
	}
	return u, nil
}
