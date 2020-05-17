// Package repo providers the model.Model implementation of the Repo entity.
package repo

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/provider"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type Repo struct {
	ID         int64  `db:"id"`
	UserID     int64  `db:"user_id"`
	ProviderID int64  `db:"provider_id"`
	HookID     int64  `db:"hook_id"`
	RepoID     int64  `db:"repo_id"`
	Enabled    bool   `db:"enabled"`
	Name       string `db:"-"`
	Href       string `db:"-"`

	User     *user.User         `db:"-"`
	Provider *provider.Provider `db:"-"`
}

type Store struct {
	model.Store

	User     *user.User
	Provider *provider.Provider
}

var (
	_ model.Model  = (*Repo)(nil)
	_ model.Binder = (*Store)(nil)

	table     = "provider_repos"
	relations = map[string]model.RelationFunc{
		"user":     model.Relation("user_id", "id"),
		"provider": model.Relation("provider_id", "id"),
	}
)

// NewStore returns a new Store for querying the provider_repos table. Each
// model passed to this function will be bound to the returned Store.
func NewStore(db *sqlx.DB, mm ...model.Model) *Store {
	s := &Store{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// Model is called along with model.Slice to convert the given slice of Repo
// models to a slice of model.Model interfaces.
func Model(rr []*Repo) func(int) model.Model {
	return func(i int) model.Model {
		return rr[i]
	}
}

// Bind the given models to the current Repo. This will only bind the model if
// they are one of the following,
//
// - *user.User
// - *provider.Provider
func (r *Repo) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			r.User = m.(*user.User)
		case *provider.Provider:
			r.Provider = m.(*provider.Provider)
		}
	}
}

func (r *Repo) SetPrimary(id int64) {
	r.ID = id
}

func (r *Repo) Primary() (string, int64) { return "id", r.ID }

func (r *Repo) IsZero() bool {
	return r == nil || r.ID == 0 &&
		r.UserID == 0 &&
		r.ProviderID == 0 &&
		r.HookID == 0 &&
		r.RepoID == 0 &&
		!r.Enabled
}

// Endpoint returns the endpoint to the current Repo, and appends any of the
// given URI parts.
func (r *Repo) Endpoint(uri ...string) string {
	endpoint := fmt.Sprintf("/repos/%v", r.ID)

	if len(uri) > 0 {
		return fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}
	return endpoint
}

func (r *Repo) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":     r.UserID,
		"provider_id": r.ProviderID,
		"hook_id":     r.HookID,
		"repo_id":     r.RepoID,
		"enabled":     r.Enabled,
	}
}

// New returns a new Repo binding any non-nil models to it from the current
// Store.
func (s *Store) New() *Repo {
	r := &Repo{
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

// Create inserts the given Repo models into the providers table.
func (s *Store) Create(rr ...*Repo) error {
	mm := model.Slice(len(rr), Model(rr))
	return errors.Err(s.Store.Create(table, mm...))
}

// Update updates the given Repo models in the providers table.
func (s *Store) Update(rr ...*Repo) error {
	mm := model.Slice(len(rr), Model(rr))
	return errors.Err(s.Store.Update(table, mm...))
}

// Delete deletes the given Repo models from the providers table.
func (s *Store) Delete(rr ...*Repo) error {
	mm := model.Slice(len(rr), Model(rr))
	return errors.Err(s.Store.Delete(table, mm...))
}

// Bind the given models to the current Repo. This will only bind the model if
// they are one of the following,
//
// - *user.User
// - *provider.Provider
func (s *Store) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		case *provider.Provider:
			s.Provider = m.(*provider.Provider)
		}
	}
}

// Get returns a single Repo model, applying each query.Option that is given.
// The model.Where option is applied to the *user.User and *provider.Provider
// bound models.
func (s *Store) Get(opts ...query.Option) (*Repo, error) {
	r := &Repo{
		User:     s.User,
		Provider: s.Provider,
	}

	opts = append([]query.Option{
		model.Where(s.User, "user_id"),
		model.Where(s.Provider, "provider_id"),
	}, opts...)

	err := s.Store.Get(r, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return r, errors.Err(err)
}

// All returns a slice Repo models, applying each query.Option that is given.
// The model.Where option is applied to the *user.User and *provider.Provider
// bound models.
func (s *Store) All(opts ...query.Option) ([]*Repo, error) {
	rr := make([]*Repo, 0)

	opts = append([]query.Option{
		model.Where(s.User, "user_id"),
		model.Where(s.Provider, "provider_id"),
	}, opts...)

	err := s.Store.All(&rr, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return rr, errors.Err(err)
}
