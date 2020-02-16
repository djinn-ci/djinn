package model

import (
	"database/sql"
	"fmt"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/types"

	"github.com/andrewpillar/query"
)

type Token struct {
	Model

	UserID int64         `db:"user_id"`
	AppID  sql.NullInt64 `db:"app_id"`
	Name   string        `db:"name"`
	Token  []byte        `db:"token"`
	Scope  types.Scope   `db:"scope"`

	User *User `db:"-"`
	App  *App  `db:"-"`
}

type TokenStore struct {
	Store

	User *User
	App  *App
}

func tokenToInterface(tt []*Token) func(i int) Interface {
	return func(i int) Interface {
		return tt[i]
	}
}

func (s TokenStore) All(opts ...query.Option) ([]*Token, error) {
	tt := make([]*Token, 0)

	baseOpts := []query.Option{
		ForUser(s.User),
		ForApp(s.App),
	}

	err := s.Store.All(&tt, TokenTable, append(baseOpts, opts...)...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, t := range tt {
		t.DB = s.DB
		t.User = s.User
		t.App = s.App
	}

	return tt, errors.Err(err)
}

func (s TokenStore) Create(tt ...*Token) error {
	models := interfaceSlice(len(tt), tokenToInterface(tt))
	return errors.Err(s.Store.Create(TokenTable, models...))
}

func (s TokenStore) Delete(tt ...*Token) error {
	models := interfaceSlice(len(tt), tokenToInterface(tt))
	return errors.Err(s.Store.Delete(TokenTable, models...))
}

func (s TokenStore) Get(opts ...query.Option) (*Token, error) {
	t := &Token{
		Model: Model{
			DB: s.DB,
		},
		User: s.User,
		App:  s.App,
	}

	baseOpts := []query.Option{
		query.Columns("*"),
		query.From(TokenTable),
		ForUser(s.User),
		ForApp(s.App),
	}

	q := query.Select(append(baseOpts, opts...)...)

	err := s.Store.Get(t, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return t, errors.Err(err)
}

func (s TokenStore) New() *Token {
	t := &Token{
		Model: Model{
			DB: s.DB,
		},
		User: s.User,
		App:  s.App,
	}

	if t.User != nil {
		t.UserID  = t.User.ID
	}

	if t.App != nil {
		t.AppID = sql.NullInt64{
			Int64: t.App.ID,
			Valid: true,
		}
	}

	return t
}

func (s TokenStore) Update(tt ...*Token) error {
	models := interfaceSlice(len(tt), tokenToInterface(tt))
	return errors.Err(s.Store.Update(TokenTable, models...))
}

func (t Token) Permissions() map[string]struct{} {
	m := make(map[string]struct{})

	spread := t.Scope.Spread()

	for _, perm := range spread {
		m[perm] = struct{}{}
	}

	return m
}

func (t Token) UIEndpoint() string {
	return fmt.Sprintf("/settings/tokens/%v", t.ID)
}

func (t Token) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":    t.UserID,
		"app_id":     t.AppID,
		"name":       t.Name,
		"token":      t.Token,
		"scope":      t.Scope,
	}
}
