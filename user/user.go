package user

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"

	"github.com/andrewpillar/query"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID          int64
	Email       string
	Username    string
	Password    []byte
	Verified    bool
	Cleanup     int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   sql.NullTime
	Permissions map[string]struct{}
}

var _ database.Model = (*User)(nil)

func (u *User) Dest() []interface{} {
	return []interface{}{
		&u.ID,
		&u.Email,
		&u.Username,
		&u.Password,
		&u.Verified,
		&u.Cleanup,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	}
}

func (*User) Bind(database.Model) {}

func (*User) Endpoint(...string) string { return "" }

func (u *User) JSON(string) map[string]interface{} {
	if u == nil {
		return nil
	}

	return map[string]interface{}{
		"id":         u.ID,
		"email":      u.Email,
		"username":   u.Username,
		"created_at": u.CreatedAt.Format(time.RFC3339),
	}
}

func (u *User) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":         u.ID,
		"email":      u.Email,
		"username":   u.Username,
		"password":   u.Password,
		"verified":   u.Verified,
		"cleanup":    u.Cleanup,
		"created_at": u.CreatedAt,
		"updated_at": u.UpdatedAt,
		"deleted_at": u.DeletedAt,
	}
}

func (u *User) SetPermission(perm string) {
	if u.Permissions == nil {
		u.Permissions = make(map[string]struct{})
	}
	u.Permissions[perm] = struct{}{}
}

type Store struct {
	database.Pool
}

var (
	_ database.Loader = (*Store)(nil)

	table      = "users"
	tokenTable = "account_tokens"

	MaxAge = 5 * 365 * 86400

	ErrAuth         = errors.New("invalid credentials")
	ErrExists       = errors.New("user exists")
	ErrTokenExpired = errors.New("token expired")
)

func WhereID(id int64) query.Option {
	return query.Options(
		query.Where("id", "=", query.Arg(id)),
		query.Where("deleted_at", "IS", query.Lit("NULL")),
	)
}

func WhereEmail(email string) query.Option {
	return query.Options(
		query.Where("email", "=", query.Arg(email)),
		query.Where("deleted_at", "IS", query.Lit("NULL")),
	)
}

func WhereUsername(username string) query.Option {
	return query.Options(
		query.Where("username", "=", query.Arg(username)),
		query.Where("deleted_at", "IS", query.Lit("NULL")),
	)
}

func WhereHandle(handle string) query.Option {
	return query.Options(
		query.Where("email", "=", query.Arg(handle)),
		query.OrWhere("username", "=", query.Arg(handle)),
		query.Where("deleted_at", "IS", query.Lit("NULL")),
	)
}

func (s Store) touchAccountToken(id int64, purpose string) (string, error) {
	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		return "", errors.Err(err)
	}

	var count int64

	q0 := query.Select(
		query.Count("*"),
		query.From(tokenTable),
		query.Where("user_id", "=", query.Arg(id)),
		query.Where("purpose", "=", query.Arg(purpose)),
	)

	if err := s.QueryRow(q0.Build(), q0.Args()...).Scan(&count); err != nil {
		return "", errors.Err(err)
	}

	tok := hex.EncodeToString(b)
	now := time.Now()

	if count == 0 {
		q := query.Insert(
			tokenTable,
			query.Columns("user_id", "token", "purpose", "created_at", "expires_at"),
			query.Values(id, tok, purpose, now, now.Add(time.Minute)),
		)

		if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
			return "", errors.Err(err)
		}
		return tok, nil
	}

	q := query.Update(
		tokenTable,
		query.Set("token", query.Arg(tok)),
		query.Set("expires_at", query.Arg(now.Add(time.Minute))),
		query.Where("user_id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return "", errors.Err(err)
	}
	return tok, nil
}

func (s Store) flushAccountToken(tok, purpose string) (int64, error) {
	var (
		id     int64
		expiry time.Time
	)

	q := query.Select(
		query.Columns("user_id", "expires_at"),
		query.From(tokenTable),
		query.Where("token", "=", query.Arg(tok)),
		query.Where("purpose", "=", query.Arg(purpose)),
	)

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&id, &expiry); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return -1, database.ErrNotFound
		}
		return -1, errors.Err(err)
	}

	q = query.Delete(
		tokenTable,
		query.Where("user_id", "=", query.Arg(id)),
		query.Where("token", "=", query.Arg(tok)),
		query.Where("purpose", "=", query.Arg(purpose)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return -1, errors.Err(err)
	}
	return id, nil
}

type Params struct {
	Email    string
	Username string
	Password string
	Cleanup  int64
}

func (s Store) Create(p Params) (*User, string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)

	if err != nil {
		return nil, "", errors.Err(err)
	}

	now := time.Now()

	q := query.Insert(
		table,
		query.Columns("email", "username", "password", "created_at"),
		query.Values(p.Email, p.Username, hash, now),
		query.Returning("id"),
	)

	var id int64

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&id); err != nil {
		if perr, ok := err.(*pgconn.PgError); ok {
			// 23505 unique_violation
			// Can occur when creating an account via OAuth2 login and a
			// username or email is already taken.
			if perr.Code == "23505" {
				return nil, "", ErrExists
			}
		}
		return nil, "", errors.Err(err)
	}

	tok, err := s.touchAccountToken(id, "verify_account")

	if err != nil {
		return nil, "", errors.Err(err)
	}

	return &User{
		ID:        id,
		Email:     p.Email,
		Username:  p.Username,
		Password:  hash,
		CreatedAt: now,
	}, tok, nil
}

func (s Store) Get(opts ...query.Option) (*User, bool, error) {
	var u User

	ok, err := s.Pool.Get(table, &u, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &u, ok, nil
}

func (s Store) All(opts ...query.Option) ([]*User, error) {
	uu := make([]*User, 0)

	new := func() database.Model {
		u := &User{}
		uu = append(uu, u)
		return u
	}

	if err := s.Pool.All(table, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return uu, nil
}

func (s Store) Load(fk, pk string, mm ...database.Model) error {
	vals := database.Values(fk, mm)

	uu, err := s.All(query.Where(pk, "IN", database.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	loaded := make([]database.Model, 0, len(uu))

	for _, u := range uu {
		loaded = append(loaded, u)
	}

	database.Bind(fk, pk, loaded, mm)
	return nil
}

func (s Store) Auth(handle, password string) (*User, error) {
	u, ok, err := s.Get(WhereHandle(handle))

	if err != nil {
		return nil, errors.Err(err)
	}

	if !ok {
		return nil, ErrAuth
	}

	if err := bcrypt.CompareHashAndPassword(u.Password, []byte(password)); err != nil {
		return nil, ErrAuth
	}
	return u, nil
}

func (s Store) RequestVerify(id int64) (string, error) {
	tok, err := s.touchAccountToken(id, "verify_account")

	if err != nil {
		return "", errors.Err(err)
	}
	return tok, nil
}

func (s Store) Verify(tok string) error {
	id, err := s.flushAccountToken(tok, "verify_account")

	if err != nil {
		return errors.Err(err)
	}

	q := query.Update(
		table, query.Set("verified", query.Arg(true)), query.Where("id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s Store) Update(id int64, p Params) error {
	opts := []query.Option{
		query.Set("email", query.Arg(p.Email)),
		query.Set("cleanup", query.Arg(p.Cleanup)),
		query.Set("updated_at", query.Arg(time.Now())),
	}

	if p.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)

		if err != nil {
			return errors.Err(err)
		}
		opts = append(opts, query.Set("password", query.Arg(hash)))
	}
	opts = append(opts, query.Where("id", "=", query.Arg(id)))

	q := query.Update(table, opts...)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s Store) UpdateEmail(tok, email string) error {
	id, err := s.flushAccountToken(tok, "email_reset")

	if err != nil {
		return errors.Err(err)
	}

	q := query.Update(
		table,
		query.Set("email", query.Arg(email)),
		query.Set("updated_at", query.Arg(time.Now())),
		query.Where("id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s Store) UpdatePassword(tok, password string) error {
	id, err := s.flushAccountToken(tok, "password_reset")

	if err != nil {
		return errors.Err(err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return errors.Err(err)
	}

	q := query.Update(
		table,
		query.Set("password", query.Arg(hash)),
		query.Set("updated_at", query.Arg(time.Now())),
		query.Where("id", "=", query.Arg(id)),
	)

	if _, err = s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s Store) ResetEmail(id int64) (string, error) {
	tok, err := s.touchAccountToken(id, "email_reset")

	if err != nil {
		return tok, errors.Err(err)
	}
	return tok, nil
}

func (s Store) ResetPassword(id int64) (string, error) {
	tok, err := s.touchAccountToken(id, "password_reset")

	if err != nil {
		return tok, errors.Err(err)
	}
	return tok, nil
}

func (s Store) Delete(id int64) error {
	q := query.Update(
		table,
		query.Set("deleted_at", query.Arg(time.Now())),
		query.Where("id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}
