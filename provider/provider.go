// Package provider provides the database.Model implementation for the Provider
// entity that represents an external provider a user has connected to
// (GitHub, GitLab, etc.).
package provider

import (
	"database/sql"
	"time"

	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/runner"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Provider is the type that represents an external Git hosting provider that
// has been connected to a user's account.
type Provider struct {
	ID             int64         `db:"id"`
	UserID         int64         `db:"user_id"`
	ProviderUserID sql.NullInt64 `db:"provider_user_id"`
	Name           string        `db:"name"`
	AccessToken    []byte        `db:"access_token"`
	RefreshToken   []byte        `db:"refresh_token"`
	Connected      bool          `db:"connected"`
	MainAccount    bool          `db:"main_account"`
	ExpiresAt      time.Time     `db:"expires_at"`
	AuthURL        string        `db:"-"`

	User *user.User `db:"-"`
}

// Store is the type for creating and modifying Provider models in the database.
type Store struct {
	database.Store

	// User is the bound user.User model. If not nil this will bind the
	// user.User model to any Provider models that are created. If not nil this
	// will append a WHERE clause on the user_id column for all SELECT queries
	// performed.
	User *user.User
}

var (
	_ database.Model  = (*Provider)(nil)
	_ database.Binder = (*Provider)(nil)

	table = "providers"
)

// NewStore returns a new Store for querying the providers table. Each database
// passed to this function will be bound to the returned Store.
func NewStore(db *sqlx.DB, mm ...database.Model) *Store {
	s := &Store{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// Model is called along with database.ModelSlice to convert the given slice of
// Provider models to a slice of database.Model interfaces.
func Model(pp []*Provider) func(int) database.Model {
	return func(i int) database.Model {
		return pp[i]
	}
}

// Select returns a query that selects the given column from the providers
// table, with each given query.Option applied to the returned query.
func Select(col string, opts ...query.Option) query.Query {
	return query.Select(query.Columns(col), append([]query.Option{query.From(table)}, opts...)...)
}

// Bind implements the database.Binder interface. This will only bind the model
// if it is a pointer to a user.User model.
func (p *Provider) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *user.User:
			p.User = v
		}
	}
}

// SetPrimary implements the database.Model interface.
func (p *Provider) SetPrimary(id int64) { p.ID = id }

// Primary implements the database.Model interface.
func (p *Provider) Primary() (string, int64) { return "id", p.ID }

// Endpoint implements the database.Model interface. This is a stub method and
// returns an empty string.
func (*Provider) Endpoint(_ ...string) string { return "" }

// IsZero implements the database.Model interface.
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

// JSON implements the database.Model interface. This is a stub method and
// returns an empty map.
func (*Provider) JSON(_ string) map[string]interface{} { return map[string]interface{}{} }

// Values implements the database.Model interface. This will return a map with
// the following values, user_id, provider_user_id, name, access_token,
// refresh_token, connected, and expires_at.
func (p *Provider) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":          p.UserID,
		"provider_user_id": p.ProviderUserID,
		"name":             p.Name,
		"access_token":     p.AccessToken,
		"refresh_token":    p.RefreshToken,
		"connected":        p.Connected,
		"main_account":     p.MainAccount,
		"expires_at":       p.ExpiresAt,
	}
}

// ToggleRepo will with add or remove a hook for the given repository hosted on
// the current provider. This will either set/unset the HookID field on the
// given Repo struct, and will toggle the Enabled field depending on whether a
// hook was added or removed.
func (p *Provider) ToggleRepo(block *crypto.Block, reg *Registry, r *Repo) error {
	cli, err := reg.Get(p.Name)

	if err != nil {
		return errors.Err(err)
	}

	tok, err := block.Decrypt(p.AccessToken)

	if err != nil {
		return errors.Err(err)
	}
	return errors.Err(cli.ToggleRepo(string(tok), r))
}

// SetCommitStatus will set the given status for the given commit sha on the
// current provider. This assumes the given commit sha is part of a merge/pull
// request.
func (p *Provider) SetCommitStatus(block *crypto.Block, reg *Registry, r *Repo, status runner.Status, url, sha string) error {
	cli, err := reg.Get(p.Name)

	if err != nil {
		return errors.Err(err)
	}

	tok, err := block.Decrypt(p.AccessToken)

	if err != nil {
		return errors.Err(err)
	}
	return errors.Err(cli.SetCommitStatus(string(tok), r, status, url, sha))
}

// Repos get's the repositories from the current provider's API endpoint. The
// given crypto.Block is used to decrypt the access token that is used to
// authenticate against the API. The given page is used to get the repositories
// on that given page.
func (p *Provider) Repos(block *crypto.Block, reg *Registry, page int64) ([]*Repo, database.Paginator, error) {
	paginator := database.Paginator{}

	if !p.Connected {
		return nil, paginator, nil
	}

	cli, err := reg.Get(p.Name)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	tok, err := block.Decrypt(p.AccessToken)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	rr, paginator, err := cli.Repos(string(tok), page)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	for i := range rr {
		rr[i].UserID = p.UserID
	}
	return rr, paginator, nil
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

// Bind implements the database.Binder interface. This will only bind the model
// if it is a pointer to a user.User model.
func (s *Store) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *user.User:
			s.User = v
		}
	}
}

// Create creates a new provider of the given name, and with the given access
// and refresh tokens. The given userId parameter should be the ID of the user's
// account from the provider we're connecting to.
func (s *Store) Create(userId int64, name string, access, refresh []byte, main, connected bool) (*Provider, error) {
	p := s.New()
	p.ProviderUserID = sql.NullInt64{
		Int64: userId,
		Valid: userId > 0,
	}
	p.Name = name
	p.AccessToken = access
	p.RefreshToken = refresh
	p.MainAccount = main
	p.Connected = connected

	err := s.Store.Create(table, p)
	return p, errors.Err(err)
}

// Update updates the provider in the database for the given id. This will set
// the userId, name, tokens, and connected status to the given values.
func (s *Store) Update(id, userId int64, name string, access, refresh []byte, main, connected bool) error {
	q := query.Update(
		table,
		query.Set("provider_user_id", query.Arg(sql.NullInt64{
			Int64: userId,
			Valid: userId > 0,
		})),
		query.Set("name", query.Arg(name)),
		query.Set("access_token", query.Arg(access)),
		query.Set("refresh_token", query.Arg(refresh)),
		query.Set("connected", query.Arg(connected)),
		query.Set("main_account", query.Arg(main)),
		query.Where("id", "=", query.Arg(id)),
	)

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// Delete deletes the given Provider models from the providers table.
func (s *Store) Delete(pp ...*Provider) error {
	mm := database.ModelSlice(len(pp), Model(pp))
	return errors.Err(s.Store.Delete(table, mm...))
}

// Get returns a single Provider model, applying each query.Option that is
// given. The database.Where option is applied to the *user.User bound database.
func (s *Store) Get(opts ...query.Option) (*Provider, error) {
	p := &Provider{
		User: s.User,
	}

	opts = append([]query.Option{
		database.Where(s.User, "user_id"),
	}, opts...)

	err := s.Store.Get(p, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return p, errors.Err(err)
}

// All returns a slice of Provider models, applying each query.Option that is
// given. The database.Where option is applied to the *user.User bound database.
func (s *Store) All(opts ...query.Option) ([]*Provider, error) {
	pp := make([]*Provider, 0)

	opts = append([]query.Option{
		database.Where(s.User, "user_id"),
	}, opts...)

	err := s.Store.All(&pp, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return pp, nil
}
