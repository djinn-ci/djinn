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

func NewStore(db *sqlx.DB, mm ...model.Model) Store {
	s := Store{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func Model(pp []*Provider) func(int) model.Model {
	return func(i int) model.Model {
		return pp[i]
	}
}

func (p *Provider) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			p.User = m.(*user.User)
		}
	}
}

func (p *Provider) Kind() string { return "provider "}

func (p *Provider) SetPrimary(id int64) {
	if p == nil {
		return
	}
	p.ID = id
}

func (p *Provider) Primary() (string, int64) {
	if p == nil {
		return "id", 0
	}
	return "id", p.ID
}

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

func (p *Provider) Values() map[string]interface{} {
	if p == nil {
		return map[string]interface{}{}
	}
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

func (s Store) New() *Provider {
	p := &Provider{
		User: s.User,
	}

	if s.User != nil {
		p.UserID = s.User.ID
	}
	return p
}

func (s *Store) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		}
	}
}

func (s Store) Create(pp ...*Provider) error {
	mm := model.Slice(len(pp), Model(pp))
	return errors.Err(s.Store.Create(table, mm...))
}

func (s Store) Update(pp ...*Provider) error {
	mm := model.Slice(len(pp), Model(pp))
	return errors.Err(s.Store.Update(table, mm...))
}

func (s Store) Delete(pp ...*Provider) error {
	mm := model.Slice(len(pp), Model(pp))
	return errors.Err(s.Store.Delete(table, mm...))
}

func (s Store) Get(opts ...query.Option) (*Provider, error) {
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

func (s Store) All(opts ...query.Option) ([]*Provider, error) {
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
