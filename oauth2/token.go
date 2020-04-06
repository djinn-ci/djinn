package oauth2

import (
	"database/sql"
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

func NewTokenStore(db *sqlx.DB, mm ...model.Model) TokenStore {
	s := TokenStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func TokenModel(tt []*Token) func(int) model.Model {
	return func(i int) model.Model {
		return tt[i]
	}
}

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

func (t Token) Kind() string { return "oauth_token" }

func (t *Token) SetPrimary(id int64) {
	t.ID = id
}

func (t Token) Primary() (string, int64) {
	return "id", t.ID
}

func (t Token) IsZero() bool {
	return t.ID == 0 &&
		t.UserID == 0 &&
		!t.AppID.Valid &&
		t.Name == "" &&
		len(t.Token) == 0 &&
		len(t.Scope) == 0 &&
		t.CreatedAt == time.Time{} &&
		t.UpdatedAt == time.Time{}
}

func (t Token) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":    t.UserID,
		"app_id":     t.AppID,
		"name":       t.Name,
		"token":      t.Token,
		"scope":      t.Scope,
		"updated_at": t.UpdatedAt,
	}
}

func (t Token) Endpoint(_ ...string) string { return "" }

func (t Token) Permissions() map[string]struct{} {
	m := make(map[string]struct{})

	spread := t.Scope.Spread()

	for _, perm := range spread {
		m[perm] = struct{}{}
	}
	return m
}

func (s TokenStore) New() *Token {
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

func (s TokenStore) All(opts ...query.Option) ([]*Token, error) {
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

func (s TokenStore) Get(opts ...query.Option) (*Token, error) {
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

func (s TokenStore) Create(tt ...*Token) error {
	mm := model.Slice(len(tt), TokenModel(tt))
	return errors.Err(s.Store.Create(tokenTable, mm...))
}

func (s TokenStore) Update(tt ...*Token) error {
	mm := model.Slice(len(tt), TokenModel(tt))
	return errors.Err(s.Store.Update(tokenTable, mm...))
}

func (s TokenStore) Delete(tt ...*Token) error {
	mm := model.Slice(len(tt), TokenModel(tt))
	return errors.Err(s.Store.Delete(tokenTable, mm...))
}
