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
	RepoID     int64  `db:"repo_id"`
	Name       string `db:"name"`
	Href       string `db:"href"`
	Enabled    bool   `db:"enabled"`

	User     *User     `db:"-",json:"-"`
	Provider *Provider `db:"-",json:"-"`
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

	r.Provider, err = providers.Find(r.ProviderID)

	return errors.Err(err)
}

func (r Repo) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":     r.UserID,
		"provider_id": r.ProviderID,
		"repo_id":     r.RepoID,
		"name":        r.Name,
		"href":        r.Href,
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

func (s RepoStore) Find(id int64) (*Repo, error) {
	r := &Repo{
		Model: Model{
			DB: s.DB,
		},
		User:     s.User,
		Provider: s.Provider,
	}

	q := query.Select(
		query.Columns("*"),
		query.From(RepoTable),
		query.Where("id", "=", id),
		ForUser(s.User),
		ForProvider(s.Provider),
	)

	err := s.Get(r, q.Build(), q.Args()...)

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
