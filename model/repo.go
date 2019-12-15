package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"

	"github.com/andrewpillar/query"
)

type Repo struct {
	Model

	UserID     int64  `db:"user_id"`
	ProviderID int64  `db:"provider_id"`
	HookID     int64  `db:"hook_id"`
	RepoID     int64  `db:"repo_id"`
	Name       string `db:"-"`
	Href       string `db:"-"`
	Enabled    bool   `db:"enabled"`

	User     *User     `db:"-" json:"-"`
	Provider *Provider `db:"-" json:"-"`
}

type RepoStore struct {
	Store

	User     *User
	Provider *Provider
}

func repoToInterface(rr []*Repo) func(i int) Interface {
	return func(i int) Interface {
		return rr[i]
	}
}

func (r *Repo) LoadProvider() error {
	var err error

	providers := &ProviderStore{
		Store: Store{
			DB: r.DB,
		},
	}

	r.Provider, err = providers.Get(query.Where("id", "=", r.ProviderID))

	return errors.Err(err)
}

func (r Repo) IsZero() bool {
	return r.Model.IsZero() &&
		r.UserID == 0 &&
		r.ProviderID == 0 &&
		r.HookID == 0 &&
		r.RepoID == 0 &&
		r.Name == ""
}

func (r Repo) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":     r.UserID,
		"provider_id": r.ProviderID,
		"hook_id":     r.HookID,
		"repo_id":     r.RepoID,
		"enabled":     r.Enabled,
	}
}

func (s RepoStore) All(opts ...query.Option) ([]*Repo, error) {
	rr := make([]*Repo, 0)

	opts = append([]query.Option{ForUser(s.User), ForProvider(s.Provider)}, opts...)

	err := s.Store.All(&rr, RepoTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, r := range rr {
		r.DB = s.DB
		r.User = s.User
		r.Provider = s.Provider
	}

	return rr, nil
}

func (s RepoStore) Create(rr ...*Repo) error {
	models := interfaceSlice(len(rr), repoToInterface(rr))

	return s.Store.Create(RepoTable, models...)
}

func (s RepoStore) Get(opts ...query.Option) (*Repo, error) {
	r := &Repo{
		Model: Model{
			DB: s.DB,
		},
		User:     s.User,
		Provider: s.Provider,
	}

	baseOpts := []query.Option{
		query.Columns("*"),
		query.From(RepoTable),
		ForUser(s.User),
		ForProvider(s.Provider),
	}

	q := query.Select(append(baseOpts, opts...)...)

	err := s.Store.Get(r, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return r, errors.Err(err)
}

func (s RepoStore) loadProvider(rr []*Repo) func(i int, p *Provider) {
	return func(i int, p *Provider) {
		r := rr[i]

		if r.ProviderID == p.ID {
			r.Provider = p
		}
	}
}

func (s RepoStore) LoadProviders(rr []*Repo) error {
	if len(rr) == 0 {
		return nil
	}

	models := interfaceSlice(len(rr), repoToInterface(rr))

	providers := ProviderStore{
		Store: s.Store,
		User:  s.User,
	}

	err := providers.Load(mapKey("provider_id", models), s.loadProvider(rr))

	return errors.Err(err)
}

func (s RepoStore) New() *Repo {
	r := &Repo{
		Model: Model{
			DB: s.DB,
		},
		User:     s.User,
		Provider: s.Provider,
	}

	if s.User != nil {
		r.UserID = s.User.ID
	}

	if s.Provider != nil {
		r.ProviderID = s.Provider.ID
	}

	return r
}

func (s RepoStore) Update(rr ...*Repo) error {
	models := interfaceSlice(len(rr), repoToInterface(rr))

	return s.Store.Update(RepoTable, models...)
}
