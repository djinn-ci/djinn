package oauth2

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
)

type Code struct {
	loaded []string

	ID        int64
	UserID    int64
	AppID     int64
	Code      string
	Scope     Scope
	ExpiresAt time.Time
}

var _ database.Model = (*Code)(nil)

func (c *Code) Primary() (string, any) { return "id", c.ID }

func (c *Code) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":         &c.ID,
		"user_id":    &c.UserID,
		"app_id":     &c.AppID,
		"code":       &c.Code,
		"scope":      &c.Scope,
		"expires_at": &c.ExpiresAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (c *Code) Params() database.Params {
	params := database.Params{
		"id":         database.ImmutableParam(c.ID),
		"user_id":    database.CreateOnlyParam(c.UserID),
		"app_id":     database.CreateOnlyParam(c.AppID),
		"code":       database.CreateOnlyParam(c.Code),
		"scope":      database.CreateOnlyParam(c.Scope),
		"expires_at": database.CreateOnlyParam(c.ExpiresAt),
	}

	if len(c.loaded) > 0 {
		params.Only(c.loaded...)
	}
	return params
}

func (*Code) Bind(database.Model)          {}
func (*Code) MarshalJSON() ([]byte, error) { return nil, nil }
func (*Code) Endpoint(...string) string    { return "" }

type CodeStore struct {
	*database.Store[*Code]
}

func NewCodeStore(pool *database.Pool) *database.Store[*Code] {
	return database.NewStore[*Code](pool, "oauth_codes", func() *Code {
		return &Code{}
	})
}

type CodeParams struct {
	User  *auth.User
	AppID int64
	Scope Scope
}

func (s CodeStore) Create(ctx context.Context, p *CodeParams) (*Code, error) {
	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		return nil, errors.Err(err)
	}

	c := Code{
		UserID:    p.User.ID,
		AppID:     p.AppID,
		Code:      hex.EncodeToString(b),
		Scope:     p.Scope,
		ExpiresAt: time.Now().Add(time.Minute * 10),
	}

	if err := s.Store.Create(ctx, &c); err != nil {
		return nil, errors.Err(err)
	}
	return &c, nil
}
