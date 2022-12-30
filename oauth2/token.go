package oauth2

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
)

type Token struct {
	ID        int64
	UserID    int64
	AppID     sql.NullInt64
	Name      string
	Token     string
	Scope     Scope
	CreatedAt time.Time
	UpdatedAt time.Time

	User *user.User
	App  *App
}

var _ database.Model = (*Token)(nil)

func (t *Token) Dest() []interface{} {
	return []interface{}{
		&t.ID,
		&t.UserID,
		&t.AppID,
		&t.Name,
		&t.Token,
		&t.Scope,
		&t.CreatedAt,
		&t.UpdatedAt,
	}
}

func (t *Token) Bind(m database.Model) {
	switch v := m.(type) {
	case *user.User:
		if t.UserID == v.ID {
			t.User = v
		}
	case *App:
		if t.AppID.Int64 == v.ID {
			t.App = v
		}
	}
}

func (*Token) JSON(_ string) map[string]interface{} { return nil }

func (t *Token) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":         t.ID,
		"user_id":    t.UserID,
		"app_id":     t.AppID,
		"name":       t.Name,
		"token":      t.Token,
		"scope":      t.Scope,
		"created_at": t.CreatedAt,
		"updated_at": t.UpdatedAt,
	}
}

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
//	build:read,write namespace:read
//
// would become the map,
//
//	map[string]struct{}{
//	    "build:read":     {},
//	    "build:write":    {},
//	    "namespace:read": {},
//	}
func (t *Token) Permissions() map[string]struct{} {
	m := make(map[string]struct{})

	spread := t.Scope.Spread()

	for _, perm := range spread {
		m[perm] = struct{}{}
	}
	return m
}

type TokenStore struct {
	database.Pool
}

var tokenTable = "oauth_tokens"

func SelectToken(col string, opts ...query.Option) query.Query {
	return query.Select(query.Columns(col), append([]query.Option{query.From(tokenTable)}, opts...)...)
}

func (s TokenStore) Get(opts ...query.Option) (*Token, bool, error) {
	var t Token

	ok, err := s.Pool.Get(tokenTable, &t, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &t, ok, nil
}

func (s TokenStore) All(opts ...query.Option) ([]*Token, error) {
	tt := make([]*Token, 0)

	new := func() database.Model {
		t := &Token{}
		tt = append(tt, t)
		return t
	}

	if err := s.Pool.All(tokenTable, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return tt, nil
}

type TokenParams struct {
	UserID int64
	AppID  int64
	Name   string
	Scope  Scope
}

func (s TokenStore) Create(p TokenParams) (*Token, error) {
	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		return nil, errors.Err(err)
	}

	appId := sql.NullInt64{
		Int64: p.AppID,
		Valid: p.AppID > 0,
	}

	tok := hex.EncodeToString(b)
	now := time.Now()

	q := query.Insert(
		tokenTable,
		query.Columns("user_id", "app_id", "name", "token", "scope", "created_at", "updated_at"),
		query.Values(p.UserID, appId, p.Name, tok, p.Scope, now, now),
		query.Returning("id"),
	)

	var id int64

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&id); err != nil {
		return nil, errors.Err(err)
	}

	return &Token{
		ID:        id,
		UserID:    p.UserID,
		AppID:     appId,
		Name:      p.Name,
		Token:     tok,
		Scope:     p.Scope,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s TokenStore) Reset(id int64) error {
	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		return errors.Err(err)
	}

	q := query.Update(
		tokenTable,
		query.Set("token", query.Arg(hex.EncodeToString(b))),
		query.Where("id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s TokenStore) Update(id int64, p TokenParams) error {
	q := query.Update(
		tokenTable,
		query.Set("name", query.Arg(p.Name)),
		query.Set("scope", query.Arg(p.Scope)),
		query.Where("id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s TokenStore) Revoke(appId int64) error {
	q := query.Delete(tokenTable, query.Where("app_id", "=", query.Arg(appId)))

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *TokenStore) Delete(id int64) error {
	q := query.Delete(tokenTable, query.Where("id", "=", query.Arg(id)))

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}
