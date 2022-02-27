package oauth2

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
)

type Code struct {
	ID        int64
	UserID    int64
	AppID     int64
	Code      string
	Scope     Scope
	ExpiresAt time.Time

	User *user.User
	App  *App
}

var _ database.Model = (*Code)(nil)

func (c *Code) Dest() []interface{} {
	return []interface{}{
		&c.ID,
		&c.UserID,
		&c.AppID,
		&c.Code,
		&c.Scope,
		&c.ExpiresAt,
	}
}

func (c *Code) Bind(m database.Model) {
	switch v := m.(type) {
	case *App:
		if c.AppID == v.ID {
			c.App = v
		}
	case *user.User:
		if c.UserID == v.ID {
			c.User = v
		}
	}
}

func (*Code) JSON(_ string) map[string]interface{} { return nil }

func (*Code) Endpoint(_ ...string) string { return "" }

func (c *Code) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":         c.ID,
		"user_id":    c.UserID,
		"app_id":     c.AppID,
		"code":       c.Code,
		"scope":      c.Scope,
		"expires_at": c.ExpiresAt,
	}
}

type CodeStore struct {
	database.Pool
}

var codeTable = "oauth_codes"

type CodeParams struct {
	UserID int64
	AppID  int64
	Scope  Scope
}

func (s CodeStore) Create(p CodeParams) (*Code, error) {
	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		return nil, errors.Err(err)
	}

	code := hex.EncodeToString(b)
	expiresAt := time.Now().Add(time.Minute * 10)

	q := query.Insert(
		codeTable,
		query.Columns("user_id", "app_id", "code", "scope", "expires_at"),
		query.Values(p.UserID, p.AppID, code, p.Scope, expiresAt),
		query.Returning("id"),
	)

	var id int64

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&id); err != nil {
		return nil, errors.Err(err)
	}

	return &Code{
		ID:        id,
		UserID:    p.UserID,
		AppID:     p.AppID,
		Code:      code,
		Scope:     p.Scope,
		ExpiresAt: expiresAt,
	}, nil
}

func (s CodeStore) Delete(id int64) error {
	q := query.Delete(codeTable, query.Where("id", "=", query.Arg(id)))

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s CodeStore) Get(opts ...query.Option) (*Code, bool, error) {
	var c Code

	ok, err := s.Pool.Get(codeTable, &c, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &c, ok, nil
}
