package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"

	"github.com/jackc/pgconn"

	"golang.org/x/crypto/bcrypt"
)

type user struct {
	*auth.User

	loaded []string

	Password  []byte
	Verified  bool
	Cleanup   int64
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt database.Null[time.Time]
}

var _ database.Model = (*user)(nil)

func Verified(u *auth.User) bool {
	if v, ok := u.RawData["verified"]; ok {
		b, ok := v.(bool)

		if ok {
			return b
		}
	}
	return false
}

func Cleanup(u *auth.User) int64 {
	if v, ok := u.RawData["cleanup"]; ok {
		n, ok := v.(int64)

		if ok {
			return n
		}
	}
	return 0
}

func authUser(u *auth.User, password []byte) error {
	hash := u.RawData["password"].([]byte)

	if err := bcrypt.CompareHashAndPassword(hash, password); err != nil {
		return auth.ErrAuth
	}
	return nil
}

func ValidatePassword(u *auth.User) webutil.ValidatorFunc {
	return func(ctx context.Context, val any) error {
		password, ok := val.([]byte)

		if !ok {
			return fmt.Errorf("user: cannot type assert %T to %T", val, []byte{})
		}
		return authUser(u, password)
	}
}

func (u *user) Primary() (string, any) { return "id", u.ID }

func (u *user) Scan(r *database.Row) error {
	u.User = &auth.User{}

	if err := u.User.Scan(r); err != nil {
		return errors.Err(err)
	}

	var ok bool

	for col, val := range u.RawData {
		switch col {
		case "password":
			if u.Password, ok = val.([]byte); !ok {
				return fmt.Errorf("user: *user.Scan: cannot type assert type %T to %T", val, u.Password)
			}
		case "verified":
			if u.Verified, ok = val.(bool); !ok {
				return fmt.Errorf("user: *user.Scan: cannot type assert type %T to %T", val, u.Verified)
			}
		case "cleanup":
			if u.Cleanup, ok = val.(int64); !ok {
				return fmt.Errorf("user: *user.Scan: cannot type assert type %T to %T", val, u.Cleanup)
			}
		case "created_at":
			if u.CreatedAt, ok = val.(time.Time); !ok {
				return fmt.Errorf("user: *user.Scan: cannot type assert type %T to %T", val, u.CreatedAt)
			}
		case "updated_at":
			if u.UpdatedAt, ok = val.(time.Time); !ok {
				return fmt.Errorf("user: *user.Scan: cannot type assert type %T to %T", val, u.UpdatedAt)
			}
		case "deleted_at":
			if val != nil {
				if u.DeletedAt, ok = val.(database.Null[time.Time]); !ok {
					return fmt.Errorf("user: *user.Scan: cannot type assert type %T to %T", val, u.DeletedAt)
				}
			}
		}
	}

	u.loaded = r.Columns
	return nil
}

func (u *user) Params() database.Params {
	params := database.Params{
		"id":         database.ImmutableParam(u.ID),
		"email":      database.CreateUpdateParam(u.Email),
		"username":   database.CreateUpdateParam(u.Username),
		"password":   database.CreateUpdateParam(u.Password),
		"verified":   database.CreateUpdateParam(u.Verified),
		"cleanup":    database.CreateUpdateParam(u.Cleanup),
		"created_at": database.CreateOnlyParam(u.CreatedAt),
		"updated_at": database.CreateUpdateParam(u.UpdatedAt),
		"deleted_at": database.UpdateOnlyParam(u.DeletedAt),
	}

	if len(u.loaded) > 0 {
		params.Only(u.loaded...)
	}
	return params
}

func (*user) Bind(database.Model)       {}
func (*user) Endpoint(...string) string { return "" }

func (u *user) SetPermission(perm string) {
	if u.Permissions == nil {
		u.Permissions = make(map[string]struct{})
	}
	u.Permissions[perm] = struct{}{}
}

func (u *user) Auth(password []byte) error {
	if err := bcrypt.CompareHashAndPassword(u.Password, password); err != nil {
		return auth.ErrAuth
	}
	return nil
}

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

type Store struct {
	*database.Store[*auth.User]
}

func Select(expr query.Expr, opts ...query.Option) query.Query {
	return query.Select(expr, append([]query.Option{query.From(table)}, opts...)...)
}

var ErrTokenExpired = errors.New("user: token expired")

const (
	table      = "users"
	tokenTable = "account_tokens"
)

func Loader(pool *database.Pool) database.Loader {
	return database.ModelLoader(pool, table, func() database.Model {
		return &auth.User{}
	})
}

func NewStore(pool *database.Pool) *database.Store[*auth.User] {
	return database.NewStore[*auth.User](pool, table, func() *auth.User {
		return &auth.User{}
	})
}

func (s Store) touchAccountToken(ctx context.Context, id int64, purpose string) (string, error) {
	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		return "", errors.Err(err)
	}

	var count int64

	q := query.Select(
		query.Count("*"),
		query.From(tokenTable),
		query.Where("user_id", "=", query.Arg(id)),
		query.Where("purpose", "=", query.Arg(purpose)),
	)

	if err := s.QueryRow(ctx, q.Build(), q.Args()...).Scan(&count); err != nil {
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

		if _, err := s.Exec(ctx, q.Build(), q.Args()...); err != nil {
			return "", errors.Err(err)
		}
		return tok, nil
	}

	q = query.Update(
		tokenTable,
		query.Set("token", query.Arg(tok)),
		query.Set("expires_at", query.Arg(now.Add(time.Minute))),
		query.Where("user_id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return "", errors.Err(err)
	}
	return tok, nil
}

type Params struct {
	Email    string
	Username string
	Password string
	Cleanup  int64
}

func (s Store) Create(ctx context.Context, p *Params) (*auth.User, error) {
	password, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)

	if err != nil {
		return nil, errors.Err(err)
	}

	u := auth.User{
		Email:    p.Email,
		Username: p.Username,
		Provider: InternalProvider,
		RawData: map[string]any{
			"email":      p.Email,
			"username":   p.Username,
			"password":   password,
			"cleanup":    1 << 30,
			"created_at": time.Now(),
		},
	}

	if err := s.Store.Create(ctx, &u); err != nil {
		err = errors.Cause(err)

		if perr, ok := err.(*pgconn.PgError); ok {
			// 23505 unique_violation
			// Can occur when creating an account via OAuth2 login and a
			// username or email is already taken.
			if perr.Code == "23505" {
				return nil, database.ErrExists
			}
		}
		return nil, errors.Err(err)
	}

	tok, err := s.touchAccountToken(ctx, u.ID, "verify_account")

	if err != nil {
		return nil, errors.Err(err)
	}

	for name, param := range u.Params() {
		u.RawData[name] = param.Value
	}
	u.RawData["account_token"] = tok

	return &u, nil
}

func (s Store) Update(ctx context.Context, u *auth.User) error {
	if err := s.Store.Update(ctx, u); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s Store) flushAccountToken(ctx context.Context, tok, purpose string) (int64, error) {
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

	if err := s.QueryRow(ctx, q.Build(), q.Args()...).Scan(&id, &expiry); err != nil {
		if errors.Is(err, database.ErrNoRows) {
			return -1, database.ErrNoRows
		}
		return -1, errors.Err(err)
	}

	q = query.Delete(
		tokenTable,
		query.Where("user_id", "=", query.Arg(id)),
		query.Where("token", "=", query.Arg(tok)),
		query.Where("purpose", "=", query.Arg(purpose)),
	)

	if _, err := s.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return -1, errors.Err(err)
	}
	return id, nil
}

func (s Store) Auth(ctx context.Context, handle, password string) (*auth.User, error) {
	u, ok, err := s.Get(ctx, WhereHandle(handle))

	if err != nil {
		return nil, errors.Err(err)
	}

	if !ok {
		return nil, auth.ErrAuth
	}

	if err := authUser(u, []byte(password)); err != nil {
		return nil, err
	}
	return u, nil
}

const InternalProvider = "djinn-ci.com/user"

func (s Store) Put(ctx context.Context, u *auth.User) (*auth.User, error) {
	if u.Provider == InternalProvider {
		return u, nil
	}

	_, ok, err := s.Get(ctx, query.Where("email", "=", query.Arg(u.Email)))

	if err != nil {
		return nil, errors.Err(err)
	}

	if ok {
		return nil, database.ErrExists
	}

	password := make([]byte, 16)
	rand.Read(password)

	u, err = s.Create(ctx, &Params{
		Email:    u.Email,
		Username: u.Username,
		Password: hex.EncodeToString(password),
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return u, nil
}

func (s Store) RequestVerify(ctx context.Context, id int64) (string, error) {
	tok, err := s.touchAccountToken(ctx, id, "verify_account")

	if err != nil {
		return "", errors.Err(err)
	}
	return tok, nil
}

func (s Store) Verify(ctx context.Context, tok string) error {
	id, err := s.flushAccountToken(ctx, tok, "verify_account")

	if err != nil {
		return errors.Err(err)
	}

	q := query.Update(
		table, query.Set("verified", query.Arg(true)), query.Where("id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s Store) UpdateEmail(ctx context.Context, tok, email string) error {
	id, err := s.flushAccountToken(ctx, tok, "email_reset")

	if err != nil {
		return errors.Err(err)
	}

	u := auth.User{
		ID:    id,
		Email: email,
		RawData: map[string]any{
			"id":    id,
			"email": email,
		},
	}

	if err := s.Update(ctx, &u); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s Store) UpdatePassword(ctx context.Context, tok, password string) error {
	id, err := s.flushAccountToken(ctx, tok, "password_reset")

	if err != nil {
		return errors.Err(err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return errors.Err(err)
	}

	u := auth.User{
		ID: id,
		RawData: map[string]any{
			"id":         id,
			"password":   hash,
			"updated_at": time.Now(),
		},
	}

	if err := s.Update(ctx, &u); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s Store) ResetEmail(ctx context.Context, u *auth.User) (string, error) {
	tok, err := s.touchAccountToken(ctx, u.ID, "email_reset")

	if err != nil {
		return tok, errors.Err(err)
	}
	return tok, nil
}

func (s Store) ResetPassword(ctx context.Context, u *auth.User) (string, error) {
	tok, err := s.touchAccountToken(ctx, u.ID, "password_reset")

	if err != nil {
		return tok, errors.Err(err)
	}
	return tok, nil
}

func (s Store) Delete(ctx context.Context, u *auth.User) error {
	deletedAt := database.Null[time.Time]{
		Elem:  time.Now(),
		Valid: true,
	}

	u.RawData["deleted_at"] = deletedAt

	err := s.Update(ctx, &auth.User{
		ID: u.ID,
		RawData: map[string]any{
			"id":         u.ID,
			"deleted_at": deletedAt,
		},
	})

	if err != nil {
		return errors.Err(err)
	}
	return nil
}
