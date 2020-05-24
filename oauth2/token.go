package oauth2

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

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

type TokenStore struct {
	model.Store

	User *user.User
	App  *App
}

var (
	_ model.Model  = (*Token)(nil)
	_ model.Binder = (*TokenStore)(nil)

	tokenTable = "oauth_tokens"
)

// NewTokenStore returns a new TokenStore for querying the oauth_tokens table.
// Each model passed to this function will be bound to the returned TokenStore.
func NewTokenStore(db *sqlx.DB, mm ...model.Model) *TokenStore {
	s := &TokenStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// TokenModel is called along with model.Slice to convert the given slice of
// Token models to a slice of model.Model interfaces.
func TokenModel(tt []*Token) func(int) model.Model {
	return func(i int) model.Model {
		return tt[i]
	}
}

// Bind the given models to the current Token. This will only bind the model if
// they are one of the following,
//
// - *app.App
// - *user.User
func (t *Token) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			t.User = m.(*user.User)
		case *App:
			t.App = m.(*App)
		}
	}
}

func (t *Token) SetPrimary(id int64) {
	t.ID = id
}

func (t *Token) Primary() (string, int64) { return "id", t.ID }

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

func (*Token) JSON(_ string) map[string]interface{} { return map[string]interface{}{} }

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
	endpoint := fmt.Sprintf("/settings/tokens/%v", t.ID)

	if len(uri) > 0 {
		return fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}
	return endpoint
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

// Bind the given models to the current Token. This will only bind the model if
// they are one of the following,
//
// - *app.App
// - *user.User
func (s *TokenStore) Bind(mm ...model.Model) {
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
// The model.Where option is applied to the bound User model and bound App
// model.
func (s *TokenStore) All(opts ...query.Option) ([]*Token, error) {
	tt := make([]*Token, 0)

	opts = append([]query.Option{
		model.Where(s.User, "user_id"),
		model.Where(s.App, "app_id"),
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

// Get returns a single Token model, applying each query.Option that is given.
// The model.Where option is applied to the bound User model and bound App
// model.
func (s *TokenStore) Get(opts ...query.Option) (*Token, error) {
	t := &Token{
		User: s.User,
		App:  s.App,
	}

	opts = append([]query.Option{
		model.Where(s.User, "user_id"),
		model.Where(s.App, "app_id"),
	}, opts...)

	err := s.Store.Get(t, tokenTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return t, errors.Err(err)
}

// Create inserts the given Token models into the oauth_tokens table.
func (s *TokenStore) Create(tt ...*Token) error {
	mm := model.Slice(len(tt), TokenModel(tt))
	return errors.Err(s.Store.Create(tokenTable, mm...))
}

// Update updates the given Token models in the oauth_tokens table.
func (s *TokenStore) Update(tt ...*Token) error {
	mm := model.Slice(len(tt), TokenModel(tt))
	return errors.Err(s.Store.Update(tokenTable, mm...))
}

// Delete deletes the given Token models from the oauth_tokens table.
func (s *TokenStore) Delete(tt ...*Token) error {
	mm := model.Slice(len(tt), TokenModel(tt))
	return errors.Err(s.Store.Delete(tokenTable, mm...))
}
