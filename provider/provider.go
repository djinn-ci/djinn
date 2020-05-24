// Package provider providers the model.Model implementation for the Provider
// entity, and implementations for the oauth2.Provider interface for the
// different Git providers that can be used to authenticate against.
package provider

import (
	"database/sql"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type Provider struct {
	ID             int64         `db:"id"`
	UserID         int64         `db:"user_id"`
	ProviderUserID sql.NullInt64 `db:"provider_user_id"`
	Name           string        `db:"name"`
	AccessToken    []byte        `db:"access_token"`
	RefreshToken   []byte        `db:"refresh_token"`
	Connected      bool          `db:"connected"`
	ExpiresAt      time.Time     `db:"expires_at"`
	AuthURL        string        `db:"-"`

	User    *user.User `db:"-"`
}

type Store struct {
	model.Store

	User *user.User
}

var (
	_ model.Model  = (*Provider)(nil)
	_ model.Binder = (*Provider)(nil)

	table = "providers"
)

// NewStore returns a new Store for querying the providers table. Each model
// passed to this function will be bound to the returned Store.
func NewStore(db *sqlx.DB, mm ...model.Model) *Store {
	s := &Store{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// Model is called along with model.Slice to convert the given slice of
// Provider models to a slice of model.Model interfaces.
func Model(pp []*Provider) func(int) model.Model {
	return func(i int) model.Model {
		return pp[i]
	}
}

// Select returns a query that selects the given column from the providers
// table, with each given query.Option applied to the returned query.
func Select(col string, opts ...query.Option) query.Query {
	return query.Select(append([]query.Option{
		query.Columns(col),
		query.From(table),
	}, opts...)...)
}

// Bind the given models to the current Provider. This will only bind the model
// if they are one of the following,
//
// - *user.User
func (p *Provider) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			p.User = m.(*user.User)
		}
	}
}

func (p *Provider) SetPrimary(id int64) {
	p.ID = id
}

func (p *Provider) Primary() (string, int64) { return "id", p.ID }

// Endpoint is a stub to fulfill the model.Model interface. It returns an empty
// string.
func (*Provider) Endpoint(_ ...string) string { return "" }

func (p *Provider) IsZero() bool {
	return p == nil || p.ID == 0 &&
		p.UserID == 0 &&
		!p.ProviderUserID.Valid &&
		p.Name == "" &&
		len(p.AccessToken) == 0 &&
		len(p.RefreshToken) == 0 &&
		!p.Connected &&
		p.ExpiresAt == time.Time{}
}

func (*Provider) JSON(_ string) map[string]interface{} { return map[string]interface{}{} }

func (p *Provider) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":          p.UserID,
		"provider_user_id": p.ProviderUserID,
		"name":             p.Name,
		"access_token":     p.AccessToken,
		"refresh_token":    p.RefreshToken,
		"connected":        p.Connected,
		"expires_at":       p.ExpiresAt,
	}
}

// New returns a new Provider binding any non-nil models to it from the current
// Store.
func (s *Store) New() *Provider {
	p := &Provider{
		User: s.User,
	}

	if s.User != nil {
		p.UserID = s.User.ID
	}
	return p
}

// Bind the given models to the current Provider. This will only bind the model
// if they are one of the following,
//
// - *user.User
func (s *Store) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		}
	}
}

// Create inserts the given Provider models into the providers table.
func (s *Store) Create(pp ...*Provider) error {
	mm := model.Slice(len(pp), Model(pp))
	return errors.Err(s.Store.Create(table, mm...))
}

// Update updates the given Provider models in the providers table.
func (s *Store) Update(pp ...*Provider) error {
	mm := model.Slice(len(pp), Model(pp))
	return errors.Err(s.Store.Update(table, mm...))
}

// Delete deletes the given Provider models from the providers table.
func (s *Store) Delete(pp ...*Provider) error {
	mm := model.Slice(len(pp), Model(pp))
	return errors.Err(s.Store.Delete(table, mm...))
}

// Get returns a single Provider model, applying each query.Option that is
// given. The model.Where option is applied to the *user.User bound model.
func (s *Store) Get(opts ...query.Option) (*Provider, error) {
	p := &Provider{
		User: s.User,
	}

	opts = append([]query.Option{
		model.Where(s.User, "user_id"),
	}, opts...)

	err := s.Store.Get(p, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return p, errors.Err(err)
}

// All returns a slice of Provider models, applying each query.Option that is
// given. The model.Where option is applied to the *user.User bound model.
func (s *Store) All(opts ...query.Option) ([]*Provider, error) {
	pp := make([]*Provider, 0)

	opts = append([]query.Option{
		model.Where(s.User, "user_id"),
	}, opts...)

	err := s.Store.All(&pp, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return pp, nil
}
