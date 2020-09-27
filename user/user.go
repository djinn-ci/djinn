// Package user provides the database implementation for the User entity.
package user

import (
	"context"
	"crypto/rand"
	"database/sql"
	"time"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"

	"golang.org/x/crypto/bcrypt"
)

// User represents a user account in the database. This will either be created
// through registration, or sign-on via an OAuth provider.
type User struct {
	ID        int64        `db:"id"`
	Email     string       `db:"email"`
	Username  string       `db:"username"`
	Password  []byte       `db:"password"`
	Verified  bool         `db:"verified"`
	Cleanup   bool         `db:"cleanup"`
	CreatedAt time.Time    `db:"created_at"`
	UpdatedAt time.Time    `db:"updated_at"`
	DeletedAt sql.NullTime `db:"deleted_at"`

	Permissions map[string]struct{} `db:"-"`
}

// Store is the type for creating and modifying User models in the database.
type Store struct {
	database.Store
}

var (
	_ database.Model  = (*User)(nil)
	_ database.Binder = (*Store)(nil)
	_ database.Loader = (*Store)(nil)

	table             = "users"
	tokenTable        = "account_tokens"
	collaboratorTable = "collaborators"

	MaxAge = 5 * 365 * 86400

	ErrAuth         = errors.New("invalid credentials")
	ErrExists       = errors.New("user exists")
	ErrTokenExpired = errors.New("token expired")
)

// NewStore returns a new Store for querying the users table. Each database
// passed to this function will be bound to the returned Store.
func NewStore(db *sqlx.DB, mm ...database.Model) *Store {
	s := &Store{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// FromContext returns the *User database from the given context value, if any.
func FromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value("user").(*User)
	return u, ok
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

// Model is called along with database.ModelSlice to convert the given slice of User
// models to a slice of database.Model interfaces.
func Model(uu []*User) func(int) database.Model {
	return func(i int) database.Model {
		return uu[i]
	}
}

// Bind implements the database.Model interface. This does nothing.
func (*User) Bind(_ ...database.Model) {}

// Endpoint implements the database.Model interface. This returns an empty
// string.
func (*User) Endpoint(_ ...string) string { return "" }

// SetPrimary implements the database.Model interface.
func (u *User) SetPrimary(id int64) { u.ID = id }

// Primary implements the database.Model interface.
func (u *User) Primary() (string, int64) { return "id", u.ID }

// IsZero implements the database.Model interface.
func (u *User) IsZero() bool {
	return u == nil || u.ID == 0 &&
		u.Email == "" &&
		u.Username == "" &&
		len(u.Password) == 0 &&
		u.CreatedAt == time.Time{} &&
		!u.DeletedAt.Valid
}

// JSON implements the database.Model interface. This will return a map with
// the values of the current user under each key. This will not include the
// password field.
func (u *User) JSON(_ string) map[string]interface{} {
	return map[string]interface{}{
		"id":         u.ID,
		"email":      u.Email,
		"username":   u.Username,
		"created_at": u.CreatedAt.Format(time.RFC3339),
	}
}

// Values implements the databae.Model interface. This will return a map with
// the following values, email, username, password, updated_at, and deleted_at.
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

func (s *Store) touchAccountToken(id int64, purpose string) ([]byte, error) {
	tok := make([]byte, 16)

	if _, err := rand.Read(tok); err != nil {
		return nil, errors.Err(err)
	}

	var count int64

	q0 := query.Select(
		query.Count("*"),
		query.From(tokenTable),
		query.Where("user_id", "=", id),
		query.Where("purpose", "=", purpose),
	)

	if err := s.DB.QueryRow(q0.Build(), q0.Args()...).Scan(&count); err != nil {
		return nil, errors.Err(err)
	}

	var q query.Query

	now := time.Now()

	if count == 0 {
		q = query.Insert(
			query.Into(tokenTable),
			query.Columns("user_id", "token", "purpose", "created_at", "expires_at"),
			query.Values(id, tok, purpose, now, now.Add(time.Minute)),
		)
	} else {
		q = query.Update(
			query.Table(tokenTable),
			query.Set("token", tok),
			query.Set("purpose", purpose),
			query.Set("expires_at", now.Add(time.Minute)),
		)
	}

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return tok, errors.Err(err)
}

func (s *Store) flushAccountToken(tok []byte, purpose string) (int64, error) {
	var (
		id     int64
		expiry time.Time
	)

	now := time.Now()

	q := query.Select(
		query.Columns("user_id", "expires_at"),
		query.From(tokenTable),
		query.Where("token", "=", tok),
		query.Where("purpose", "=", purpose),
	)

	if err := s.DB.QueryRow(q.Build(), q.Args()...).Scan(&id, &expiry); err != nil {
		if err == sql.ErrNoRows {
			return 0, database.ErrNotFound
		}
		return 0, errors.Err(err)
	}

	if id == 0 {
		return 0, database.ErrNotFound
	}

	q1 := query.Delete(
		query.From(tokenTable),
		query.Where("user_id", "=", id),
		query.Where("token", "=", tok),
		query.Where("purpose", "=", purpose),
	)

	if _, err := s.DB.Exec(q1.Build(), q1.Args()...); err != nil {
		return 0, errors.Err(err)
	}

	if expiry.Before(now) {
		return 0, ErrTokenExpired
	}
	return id, nil
}

// Bind implements the database.Model interface. This does nothing.
func (s *Store) Bind(_ ...database.Model) {}

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
// of given vals. Each database is loaded individually via a call to the given
// load callback.
func (s *Store) Load(key string, vals []interface{}, load database.LoaderFunc) error {
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

// Create creates a new user with the given email, username and password. The
// given password is hashed via bcrypt using the default cost.
func (s *Store) Create(email, username string, password []byte) (*User, []byte, error) {
	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)

	if err != nil {
		return nil, nil, errors.Err(err)
	}

	u := s.New()
	u.Email = email
	u.Username = username
	u.Password = hash
	u.UpdatedAt = time.Now()

	if err := s.Store.Create(table, u); err != nil {
		return nil, nil, errors.Err(err)
	}

	tok, err := s.touchAccountToken(u.ID, "verify_account")
	return u, tok, errors.Err(err)
}

func (s *Store) RequestVerify(id int64) ([]byte, error) {
	tok, err := s.touchAccountToken(id, "verify_account")
	return tok, errors.Err(err)
}

func (s *Store) Verify(tok []byte) error {
	id, err := s.flushAccountToken(tok, "verify_account")

	if err != nil {
		return errors.Err(err)
	}

	q := query.Update(
		query.Table(table),
		query.Set("verified", true),
		query.Where("id", "=", id),
	)

	_, err = s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// Update sets the email, cleanup, and password fields for the given user to
// the given values. If the given password is nil, then this will not be
// updated, otherwise a new hash is generated for it.
func (s *Store) Update(id int64, email string, cleanup bool, password []byte) error {
	opts := []query.Option{
		query.Table(table),
		query.Set("email", email),
		query.Set("cleanup", cleanup),
		query.Set("updated_at", time.Now()),
	}

	if password != nil {
		hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)

		if err != nil {
			return errors.Err(err)
		}

		opts = append(opts, query.Set("password", hash))
	}

	q := query.Update(opts...)

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// Delete the user with the given id. This will set the deleted_at field in the
// table to the time at which this method was called.
func (s *Store) Delete(id int64, password []byte) error {
	u, err := s.Get(query.Where("id", "=", id), query.WhereRaw("deleted_at", "IS", "NULL"))

	if err != nil {
		return errors.Err(err)
	}

	if err := bcrypt.CompareHashAndPassword(u.Password, password); err != nil {
		return ErrAuth
	}

	q := query.Update(
		query.Table(table),
		query.Set("deleted_at", time.Now()),
	)

	_, err = s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

func (s *Store) UpdatePassword(tok, password []byte) error {
	id, err := s.flushAccountToken(tok, "password_reset")

	if err != nil {
		return errors.Err(err)
	}

	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)

	if err != nil {
		return errors.Err(err)
	}

	q := query.Update(
		query.Table(table),
		query.Set("password", hash),
		query.Where("id", "=", id),
	)

	_, err = s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

func (s *Store) ResetPassword(id int64) ([]byte, error) {
	tok, err := s.touchAccountToken(id, "password_reset")
	return tok, errors.Err(err)
}

// Get returns a single User database, applying each query.Option that is given.
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
	u, err := s.Get(WhereHandle(handle), query.WhereRaw("deleted_at", "IS", "NULL"))

	if err != nil {
		return nil, errors.Err(err)
	}

	if u.IsZero() {
		return nil, ErrAuth
	}

	if err := bcrypt.CompareHashAndPassword(u.Password, []byte(password)); err != nil {
		return nil, ErrAuth
	}
	return u, nil
}
