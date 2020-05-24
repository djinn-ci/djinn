// Package user provides the model implementation for the User entity.
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

// NewStore returns a new Store for querying the users table. Each model
// passed to this function will be bound to the returned Store.
func NewStore(db *sqlx.DB, mm ...model.Model) *Store {
	s := &Store{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// Select returns a query that selects the given column from the users table,
// with each given query.Option applied to the returned query.
func Select(col string, opts ...query.Option) query.Query {
	return query.Select(append([]query.Option{
		query.Columns(col),
		query.From(table),
	}, opts...)...)
}

// WhereHandle returns a query.Option that when applied to a query will add two
// WHERE clauses that will check the given handle against the email column or
// the username column.
func WhereHandle(handle string) query.Option {
	return query.Options(
		query.Where("email", "=", handle),
		query.OrWhere("username", "=", handle),
	)
}

// Model is called along with model.Slice to convert the given slice of User
// models to a slice of model.Model interfaces.
func Model(uu []*User) func(int) model.Model {
	return func(i int) model.Model {
		return uu[i]
	}
}

// Bind is a stub method to satisfy the model.Model interface.
func (*User) Bind(_ ...model.Model) {}

// Endpoint is a stub method to satisy the model.Model interface.
func (*User) Endpoint(_ ...string) string { return "" }

func (u *User) SetPrimary(id int64) {
	u.ID = id
}

func (u *User) Primary() (string, int64) {
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

func (u *User) JSON(addr string) map[string]interface{} {
	return map[string]interface{}{
		"id":         u.ID,
		"email":      u.Email,
		"username":   u.Username,
		"created_at": u.CreatedAt.Format(time.RFC3339),
		"updated_at": u.UpdatedAt.Format(time.RFC3339),
		"url":        addr + "/user",
	}
}

func (u *User) Values() map[string]interface{} {
	return map[string]interface{}{
		"email":      u.Email,
		"username":   u.Username,
		"password":   u.Password,
		"updated_at": u.UpdatedAt,
		"deleted_at": u.DeletedAt,
	}
}

// SetPermission set's the given permission in the underlying Permissions map
// of the current User. If the map is nil then it will be initialized.
func (u *User) SetPermission(perm string) {
	if u.Permissions == nil {
		u.Permissions = make(map[string]struct{})
	}
	u.Permissions[perm] = struct{}{}
}

// Bind is a stub method to statisy the model.Binder interface.
func (s *Store) Bind(_ ...model.Model) {}

// All returns a slice of User models, applying each query.Option that is
// given.
func (s *Store) All(opts ...query.Option) ([]*User, error) {
	uu := make([]*User, 0)

	err := s.Store.All(&uu, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return uu, errors.Err(err)
}

// Load loads in a slice of User models where the given key is in the list
// of given vals. Each model is loaded individually via a call to the given
// load callback.
func (s *Store) Load(key string, vals []interface{}, load model.LoaderFunc) error {
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

// New returns a new zero-value User model.
func (*Store) New() *User { return &User{} }

// Create inserts the given User models into the users table.
func (s *Store) Create(uu ...*User) error {
	models := model.Slice(len(uu), Model(uu))
	return errors.Err(s.Store.Create(table, models...))
}

// Update updates the given Build models in the users table.
func (s *Store) Update(uu ...*User) error {
	models := model.Slice(len(uu), Model(uu))
	return errors.Err(s.Store.Update(table, models...))
}

// Get returns a single User model, applying each query.Option that is given.
func (s *Store) Get(opts ...query.Option) (*User, error) {
	u := &User{}

	err := s.Store.Get(u, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return u, errors.Err(err)
}

// Auth looks up the user by the given handle, and checks that the given
// password matches the hash in the database.
func (s *Store) Auth(handle, password string) (*User, error) {
	u, err := s.Get(WhereHandle(handle))

	if err != nil {
		return u, errors.Err(err)
	}

	if err := bcrypt.CompareHashAndPassword(u.Password, []byte(password)); err != nil {
		return u, ErrAuth
	}
	return u, nil
}
