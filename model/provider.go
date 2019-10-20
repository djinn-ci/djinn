package model

import (
	"database/sql"
	"time"

	"github.com/andrewpillar/thrall/errors"

	"github.com/andrewpillar/query"
)

type Provider struct {
	Model

	UserID       int64     `db:"user_id"`
	Name         string    `db:"name"`
	AccessToken  []byte    `db:"access_token"`
	RefreshToken []byte    `db:"refresh_token"`
	ExpiresAt    time.Time `db:"expires_at"`

	User *User `db:"-"`
}

type ProviderStore struct {
	Store

	User *User
}

func providerToInterface(pp []*Provider) func(i int) Interface {
	return func(i int) Interface {
		return pp[i]
	}
}

func (p Provider) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":       p.UserID,
		"name":          p.Name,
		"access_token":  p.AccessToken,
		"refresh_token": p.RefreshToken,
		"expires_at":    p.ExpiresAt,
		"created_at":    p.CreatedAt,
		"updated_at":    p.UpdatedAt,
	}
}

func (s ProviderStore) Create(pp ...*Provider) error {
	models := interfaceSlice(len(pp), providerToInterface(pp))

	return errors.Err(s.Store.Create(ProviderTable, models...))
}

func (s ProviderStore) FindByName(name string) (*Provider, error) {
	p := &Provider{
		Model: Model{
			DB: s.DB,
		},
		User: s.User,
	}

	q := query.Select(
		query.Columns("*"),
		query.From(ProviderTable),
		query.Where(name, "=", name),
		ForUser(s.User),
	)

	err := s.Get(p, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return p, errors.Err(err)
}

func (s ProviderStore) New() *Provider {
	p := &Provider{
		Model: Model{
			DB: s.DB,
		},
		User: s.User,
	}

	if s.User != nil {
		p.UserID = s.User.ID
	}

	return p
}

func (s ProviderStore) Update(pp ...*Provider) error {
	models := interfaceSlice(len(pp), providerToInterface(pp))

	return errors.Err(s.Store.Update(ProviderTable, models...))
}
