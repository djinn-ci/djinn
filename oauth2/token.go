package oauth2

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"

	"github.com/andrewpillar/query"
)

type Token struct {
	loaded []string

	ID        int64
	UserID    int64
	AppID     database.Null[int64]
	Name      string
	Token     string
	Scope     Scope
	CreatedAt time.Time
	UpdatedAt time.Time

	App *App
}

var _ database.Model = (*Token)(nil)

func (t *Token) Primary() (string, any) { return "id", t.ID }

func (t *Token) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":         &t.ID,
		"user_id":    &t.UserID,
		"app_id":     &t.AppID,
		"name":       &t.Name,
		"token":      &t.Token,
		"scope":      &t.Scope,
		"created_at": &t.CreatedAt,
		"updated_at": &t.UpdatedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (t *Token) Params() database.Params {
	params := database.Params{
		"id":         database.ImmutableParam(t.ID),
		"user_id":    database.CreateOnlyParam(t.UserID),
		"app_id":     database.CreateOnlyParam(t.AppID),
		"name":       database.CreateUpdateParam(t.Name),
		"token":      database.CreateOnlyParam(t.Token),
		"scope":      database.CreateUpdateParam(t.Scope),
		"created_at": database.CreateOnlyParam(t.CreatedAt),
		"updated_at": database.UpdateOnlyParam(t.UpdatedAt),
	}

	if len(t.loaded) > 0 {
		params.Only(t.loaded...)
	}
	return params
}

func (t *Token) Bind(m database.Model) {
	if v, ok := m.(*App); ok {
		if t.AppID.Elem == v.ID {
			t.App = v
		}
	}
}

func (*Token) MarshalJSON() ([]byte, error) { return nil, nil }

func (t *Token) Endpoint(elems ...string) string {
	if len(elems) > 0 {
		return "/settings/tokens/" + strconv.FormatInt(t.ID, 10) + "/" + strings.Join(elems, "/")
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
	*database.Store[*Token]
}

func NewTokenStore(pool *database.Pool) *database.Store[*Token] {
	return database.NewStore[*Token](pool, "oauth_tokens", func() *Token {
		return &Token{}
	})
}

type TokenParams struct {
	User  *auth.User
	AppID int64
	Name  string
	Scope Scope
}

func (s TokenStore) Create(ctx context.Context, p *TokenParams) (*Token, error) {
	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		return nil, errors.Err(err)
	}

	now := time.Now()

	t := Token{
		UserID: p.User.ID,
		AppID: database.Null[int64]{
			Elem:  p.AppID,
			Valid: p.AppID > 0,
		},
		Name:      p.Name,
		Token:     hex.EncodeToString(b),
		Scope:     p.Scope,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.Store.Create(ctx, &t); err != nil {
		return nil, errors.Err(err)
	}
	return &t, nil
}

func (s TokenStore) Reset(ctx context.Context, t *Token) error {
	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		return errors.Err(err)
	}

	t.Token = hex.EncodeToString(b)
	t.UpdatedAt = time.Now()

	if err := s.Update(ctx, t); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s TokenStore) Revoke(ctx context.Context, t *Token) error {
	q := query.Delete("oauth_tokens", query.Where("app_id", "=", query.Arg(t.AppID)))

	if _, err := s.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}
