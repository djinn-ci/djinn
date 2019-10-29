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
	Connected    bool      `db:"connected"`
	ExpiresAt    time.Time `db:"expires_at"`

	User  *User   `db:"-"`
	Repos []*Repo `db:"-"`
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
		"connected":     p.Connected,
		"expires_at":    p.ExpiresAt,
		"created_at":    p.CreatedAt,
		"updated_at":    p.UpdatedAt,
	}
}

func (s ProviderStore) All(opts ...query.Option) ([]*Provider, error) {
	pp := make([]*Provider, 0)

	opts = append([]query.Option{ForUser(s.User)}, opts...)

	err := s.Store.All(&pp, ProviderTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, p := range pp {
		p.DB = s.DB
		p.User = s.User
	}

	return pp, errors.Err(err)
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

func (s ProviderStore) Load(ids []interface{}, load func(i int, p *Provider)) error {
	if len(ids) == 0 {
		return nil
	}

	pp, err := s.All(query.Where("id", "IN", ids...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range ids {
		for _, p := range pp {
			load(i, p)
		}
	}

	return nil
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