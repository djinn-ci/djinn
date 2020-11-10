package oauth2

import (
	"context"
	"crypto/rand"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Token is the type that represents an OAuth Token in the database.
type Token struct {
	ID        int64         `db:"id"`
	UserID    int64         `db:"user_id"`
	AppID     sql.NullInt64 `db:"app_id"`
	Name      string        `db:"name"`
	Token     []byte        `db:"token"`
	Scope     Scope         `db:"scope"`
	CreatedAt time.Time     `db:"created_at"`
	UpdatedAt time.Time     `db:"updated_at"`

	User *user.User `db:"-"`
	App  *App       `db:"-"`
}

// TokenStore is the type for creating and modifying Token models in the
// database.
type TokenStore struct {
	database.Store

	// User is the bound user.User model. If not nil this will bind the
	// user.User model to any Token models that are created. If not nil this
	// will append a WHERE clause on the user_id column for all SELECT queries
	// performed.
	User *user.User

	// App is the bound App model. If not nil this will bind the App model to
	// any Token models that are created. If not nil this will append a WHERE
	// clause on the app_id column for all SELECT queries performed.
	App *App
}

var (
	_ database.Model  = (*Token)(nil)
	_ database.Binder = (*TokenStore)(nil)

	tokenTable = "oauth_tokens"
)

// NewTokenStore returns a new TokenStore for querying the oauth_tokens table.
// Each database passed to this function will be bound to the returned TokenStore.
func NewTokenStore(db *sqlx.DB, mm ...database.Model) *TokenStore {
	s := &TokenStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// TokenFromContext returns the Token model from the given context, if any.
func TokenFromContext(ctx context.Context) (*Token, bool) {
	t, ok := ctx.Value("token").(*Token)
	return t, ok
}

// TokenModel is called along with database.ModelSlice to convert the given slice of
// Token models to a slice of database.Model interfaces.
func TokenModel(tt []*Token) func(int) database.Model {
	return func(i int) database.Model {
		return tt[i]
	}
}

// SelectToken returns SELECT query that will select the given column from the
// oauth_tokens table with the given query options applied.
func SelectToken(col string, opts ...query.Option) query.Query {
	return query.Select(query.Columns(col), append([]query.Option{query.From(tokenTable)}, opts...)...)
}

// Bind implements the database.Binder interface. This will only bind the model
// if it is a pointer to either a user.User model or an App model.
func (t *Token) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			t.User = m.(*user.User)
		case *App:
			t.App = m.(*App)
		}
	}
}

// SetPrimary implements the database.Model interface.
func (t *Token) SetPrimary(id int64) { t.ID = id }

// Primary implements the database.Model interface.
func (t *Token) Primary() (string, int64) { return "id", t.ID }

// IsZero implements the database.Model interface.
func (t *Token) IsZero() bool {
	return t.ID == 0 &&
		t.UserID == 0 &&
		!t.AppID.Valid &&
		t.Name == "" &&
		len(t.Token) == 0 &&
		len(t.Scope) == 0 &&
		t.CreatedAt == time.Time{} &&
		t.UpdatedAt == time.Time{}
}

// JSON implements the database.Model interface. This is a stub method and
// returns an empty map.
func (*Token) JSON(_ string) map[string]interface{} { return map[string]interface{}{} }

// Values implements the database.Model interface. This will return a map with
// the following values, user_id, app_id, name, token, scope, and updated_at.
func (t *Token) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":    t.UserID,
		"app_id":     t.AppID,
		"name":       t.Name,
		"token":      t.Token,
		"scope":      t.Scope,
		"updated_at": t.UpdatedAt,
	}
}

// Endpoint returns the endpoint for the current Token with the appended URI
// parts.
func (t *Token) Endpoint(uri ...string) string {
	if len(uri) > 0 {
		return "/settings/tokens/" + strconv.FormatInt(t.ID, 10) + "/" + strings.Join(uri, "/")
	}
	return "/settings/tokens/" + strconv.FormatInt(t.ID, 10)
}

// Permissions turns the current Token's permission into a map. This will
// spread out the Token's scope into a space delimited string of
// resource:permission values. Each part of the space delimited string will
// be a key in the returned map, for example,
//
//   build:read,write namespace:read
//
// would become the map,
//
//   map[string]struct{}{
//       "build:read":     {},
//       "build:write":    {},
//       "namespace:read": {},
//   }
func (t *Token) Permissions() map[string]struct{} {
	m := make(map[string]struct{})

	spread := t.Scope.Spread()

	for _, perm := range spread {
		m[perm] = struct{}{}
	}
	return m
}

// New returns a new Token binding any non-nil models to it from the current
// TokenStore.
func (s *TokenStore) New() *Token {
	t := &Token{
		User: s.User,
		App:  s.App,
	}

	if s.User != nil {
		t.UserID = s.User.ID
	}

	if s.App != nil {
		t.AppID = sql.NullInt64{
			Int64: s.App.ID,
			Valid: true,
		}
	}
	return t
}

// Bind implements the database.Binder interface. This will only bind the model
// if it is a pointer to either a user.User model or an App model.
func (s *TokenStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		case *App:
			s.App = m.(*App)
		}
	}
}

// All returns a slice Token models, applying each query.Option that is given.
// The database.Where option is applied to the bound User database and bound App
// database.
func (s *TokenStore) All(opts ...query.Option) ([]*Token, error) {
	tt := make([]*Token, 0)

	opts = append([]query.Option{
		database.Where(s.User, "user_id"),
		database.Where(s.App, "app_id"),
	}, opts...)

	err := s.Store.All(&tt, tokenTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, t := range tt {
		t.User = s.User
		t.App = s.App
	}
	return tt, errors.Err(err)
}

// Get returns a single Token database, applying each query.Option that is given.
// The database.Where option is applied to the bound User database and bound App
// database.
func (s *TokenStore) Get(opts ...query.Option) (*Token, error) {
	t := &Token{
		User: s.User,
		App:  s.App,
	}

	opts = append([]query.Option{
		database.Where(s.User, "user_id"),
		database.Where(s.App, "app_id"),
	}, opts...)

	err := s.Store.Get(t, tokenTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return t, errors.Err(err)
}

// Create creates a new token with the given name, and scopes.
func (s *TokenStore) Create(name string, sc Scope) (*Token, error) {
	var err error

	t := s.New()
	t.Name = name
	t.Token = make([]byte, 16)
	t.Scope = sc

	if _, err := rand.Read(t.Token); err != nil {
		return t, errors.Err(err)
	}

	err = s.Store.Create(tokenTable, t)
	return t, errors.Err(err)
}

// Reset generates a new token value for the Token of the given id.
func (s *TokenStore) Reset(id int64) error {
	token := make([]byte, 16)

	if _, err := rand.Read(token); err != nil {
		return errors.Err(err)
	}

	q := query.Update(
		tokenTable,
		query.Set("token", query.Arg(token)),
		query.Where("id", "=", query.Arg(id)),
	)

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// Update updates a Token for the given id with the new name and scope.
func (s *TokenStore) Update(id int64, name string, sc Scope) error {
	q := query.Update(
		tokenTable,
		query.Set("name", query.Arg(name)),
		query.Set("scope", query.Arg(sc)),
		query.Where("id", "=", query.Arg(id)),
	)

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// Revoke deletes all of the tokens for the given appId.
func (s *TokenStore) Revoke(appId int64) error {
	q := query.Delete(tokenTable, query.Where("app_id", "=", query.Arg(appId)))

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// Delete deletes the token of the given ids from the database.
func (s *TokenStore) Delete(ids ...int64) error {
	mm := make([]database.Model, 0, len(ids))

	for _, id := range ids {
		mm = append(mm, &Token{ID: id})
	}
	return errors.Err(s.Store.Delete(tokenTable, mm...))
}
