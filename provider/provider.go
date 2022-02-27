package provider

import (
	"context"
	"database/sql"

	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/runner"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"

	"github.com/jackc/pgx/v4"
)

type Provider struct {
	aesgcm  *crypto.AESGCM
	clients *Registry

	ID             int64
	UserID         int64
	ProviderUserID sql.NullInt64
	Name           string
	AccessToken    []byte
	RefreshToken   []byte
	Connected      bool
	MainAccount    bool
	ExpiresAt      sql.NullTime
	AuthURL        string

	User *user.User
}

var _ database.Model = (*Provider)(nil)

func (p *Provider) Dest() []interface{} {
	return []interface{}{
		&p.ID,
		&p.UserID,
		&p.ProviderUserID,
		&p.Name,
		&p.AccessToken,
		&p.RefreshToken,
		&p.Connected,
		&p.MainAccount,
		&p.ExpiresAt,
	}
}

func (p *Provider) Bind(m database.Model) {
	if v, ok := m.(*user.User); ok {
		if p.UserID == v.ID {
			p.User = v
		}
	}
}

func (*Provider) Endpoint(_ ...string) string { return "" }

func (*Provider) JSON(_ string) map[string]interface{} { return nil }

func (p *Provider) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":               p.ID,
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

func (p *Provider) ToggleRepo(r *Repo) error {
	if p.aesgcm == nil {
		return crypto.ErrNilAESGCM
	}

	if p.clients == nil {
		return ErrNilRegistry
	}

	cli, err := p.clients.Get(p.Name)

	if err != nil {
		return errors.Err(err)
	}

	tok, err := p.aesgcm.Decrypt(p.AccessToken)

	if err != nil {
		return errors.Err(err)
	}

	if err := cli.ToggleRepo(string(tok), r); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (p *Provider) SetCommitStatus(r *Repo, status runner.Status, url, sha string) error {
	if p.aesgcm == nil {
		return crypto.ErrNilAESGCM
	}

	if p.clients == nil {
		return ErrNilRegistry
	}

	cli, err := p.clients.Get(p.Name)

	if err != nil {
		return errors.Err(err)
	}

	tok, err := p.aesgcm.Decrypt(p.AccessToken)

	if err != nil {
		return errors.Err(err)
	}

	if err := cli.SetCommitStatus(string(tok), r, status, url, sha); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (p *Provider) Repos(page int64) ([]*Repo, database.Paginator, error) {
	if p.aesgcm == nil {
		return nil, database.Paginator{}, crypto.ErrNilAESGCM
	}

	if p.clients == nil {
		return nil, database.Paginator{}, ErrNilRegistry
	}

	var paginator database.Paginator

	if !p.Connected {
		return nil, paginator, nil
	}

	cli, err := p.clients.Get(p.Name)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	tok, err := p.aesgcm.Decrypt(p.AccessToken)

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

type Store struct {
	database.Pool

	AESGCM *crypto.AESGCM

	Clients *Registry

	Cache RepoCache
}

type Params struct {
	UserID         int64
	ProviderUserID int64
	Name           string
	AccessToken    string
	RefreshToken   string
	Connected      bool
	MainAccount    bool
}

var table = "providers"

func Select(col string, opts ...query.Option) query.Query {
	return query.Select(query.Columns(col), append([]query.Option{query.From(table)}, opts...)...)
}

func (s *Store) Create(p Params) (*Provider, error) {
	if s.AESGCM == nil {
		return nil, crypto.ErrNilAESGCM
	}

	providerUserId := sql.NullInt64{
		Int64: p.ProviderUserID,
		Valid: p.ProviderUserID > 0,
	}

	access, err := s.AESGCM.Encrypt([]byte(p.AccessToken))

	if err != nil {
		return nil, errors.Err(err)
	}

	refresh, err := s.AESGCM.Encrypt([]byte(p.RefreshToken))

	if err != nil {
		return nil, errors.Err(err)
	}

	q := query.Insert(
		table,
		query.Columns("user_id", "provider_user_id", "name", "access_token", "refresh_token", "connected", "main_account"),
		query.Values(p.UserID, providerUserId, p.Name, access, refresh, p.Connected, p.MainAccount),
		query.Returning("id"),
	)

	var id int64

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&id); err != nil {
		return nil, errors.Err(err)
	}

	return &Provider{
		aesgcm:         s.AESGCM,
		clients:        s.Clients,
		ID:             id,
		UserID:         p.UserID,
		ProviderUserID: providerUserId,
		Name:           p.Name,
		AccessToken:    access,
		RefreshToken:   refresh,
		Connected:      p.Connected,
		MainAccount:    p.MainAccount,
	}, nil
}

func (s *Store) Update(id int64, p Params) error {
	if s.AESGCM == nil {
		return crypto.ErrNilAESGCM
	}

	providerUserId := sql.NullInt64{
		Int64: p.ProviderUserID,
		Valid: p.ProviderUserID > 0,
	}

	var (
		access  []byte
		refresh []byte
	)

	if p.AccessToken != "" {
		b, err := s.AESGCM.Encrypt([]byte(p.AccessToken))

		if err != nil {
			return errors.Err(err)
		}
		access = b
	}

	if p.RefreshToken != "" {
		b, err := s.AESGCM.Encrypt([]byte(p.RefreshToken))

		if err != nil {
			return errors.Err(err)
		}
		refresh = b
	}

	q := query.Update(
		table,
		query.Set("provider_user_id", query.Arg(providerUserId)),
		query.Set("name", query.Arg(p.Name)),
		query.Set("access_token", query.Arg(access)),
		query.Set("refresh_token", query.Arg(refresh)),
		query.Set("connected", query.Arg(p.Connected)),
		query.Set("main_account", query.Arg(p.MainAccount)),
		query.Where("id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *Store) DeleteAll(ids ...int64) error {
	if len(ids) == 0 {
		return nil
	}

	vals := make([]interface{}, 0, len(ids))

	for _, id := range ids {
		vals = append(vals, id)
	}

	q := query.Delete(table, query.Where("id", "IN", query.List(vals...)))

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

// Delete deletes the provider with the given name for the given user. This
// will also delete all of the repos for that provider too.
func (s *Store) Delete(ctx context.Context, name string, userId int64) error {
	tx, err := s.Begin(ctx)

	if err != nil {
		return errors.Err(err)
	}

	defer tx.Rollback(ctx)

	q := query.Delete(
		repoTable,
		query.Where("provider_id", "IN", query.Select(
			query.Columns("id"),
			query.From(table),
			query.Where("user_id", "=", query.Arg(userId)),
			query.Where("name", "=", query.Arg(name)),
		)),
	)

	if _, err := tx.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}

	q = query.Delete(
		table,
		query.Where("user_id", "=", query.Arg(userId)),
		query.Where("name", "=", query.Arg(name)),
	)

	if _, err := tx.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *Store) Connected(userId int64) (bool, error) {
	var connected bool

	q := query.Select(
		query.Columns("connected"),
		query.From(table),
		query.Where("user_id", "=", query.Arg(userId)),
		query.Where("connected", "=", query.Arg(true)),
	)

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&connected); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return false, errors.Err(err)
		}
	}
	return connected, nil
}

func (s *Store) Get(opts ...query.Option) (*Provider, bool, error) {
	var p Provider

	ok, err := s.Pool.Get(table, &p, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}

	p.aesgcm = s.AESGCM
	p.clients = s.Clients
	return &p, ok, nil
}

func (s *Store) All(opts ...query.Option) ([]*Provider, error) {
	pp := make([]*Provider, 0)

	new := func() database.Model {
		p := &Provider{
			aesgcm:  s.AESGCM,
			clients: s.Clients,
		}
		pp = append(pp, p)
		return p
	}

	if err := s.Pool.All(table, new, opts...); err != nil {
		return nil, errors.Err(err)
	}

	return pp, nil
}

// LoadRepos will load all of the user's repositories for the provider with the
// given name. If no name is given, then the first connected provider that is
// found whilst sorting lexically is used. This will first check the underlying
// cache for repos, if the cache is empty, then the API for the respective
// provider is hit, then the repos are cached.
func (s *Store) LoadRepos(userId int64, name string, page int64) (*Provider, []*Repo, database.Paginator, error) {
	var paginator database.Paginator

	opts := []query.Option{
		query.Where("user_id", "=", query.Arg(userId)),
	}

	if name != "" {
		opts = append(opts, query.Where("name", "=", query.Arg(name)))
	} else {
		opts = append(opts,
			query.Where("connected", "=", query.Arg(true)),
			query.OrderAsc("name"),
		)
	}

	pp, err := s.All(opts...)

	if err != nil {
		return nil, nil, paginator, errors.Err(err)
	}

	// No providers connected, so exit early.
	if len(pp) == 0 {
		return nil, nil, paginator, nil
	}

	var mainProvider *Provider

	providers := make(map[int64]*Provider)

	for _, p := range pp {
		if p.MainAccount {
			if mainProvider == nil {
				mainProvider = p
			}
		}
		providers[p.ProviderUserID.Int64] = p
	}

	rr, paginator, err := s.Cache.Get(mainProvider, page)

	if err != nil {
		return nil, nil, paginator, errors.Err(err)
	}

	if len(rr) == 0 {
		rr, paginator, err = mainProvider.Repos(page)

		if err != nil {
			return nil, nil, paginator, errors.Err(err)
		}

		if len(rr) > 0 {
			if err := s.Cache.Put(mainProvider, rr, paginator); err != nil {
				return nil, nil, paginator, errors.Err(err)
			}
		}
	}

	q := query.Select(
		query.Columns("id", "provider_id", "repo_id"),
		query.From(repoTable),
		query.Where("user_id", "=", query.Arg(userId)),
		query.Where("enabled", "=", query.Arg(true)),
	)

	rows, err := s.Query(q.Build(), q.Args()...)

	if err != nil {
		return nil, nil, paginator, errors.Err(err)
	}

	type enabledkey struct {
		providerId int64
		repoId     int64
	}

	// Store the IDs of enabled repos under a key that is made up of the
	// provider ID, and the ID of the repo from the provider's side.
	enabled := make(map[enabledkey]int64)

	var (
		id  int64
		key enabledkey
	)

	for rows.Next() {
		if err := rows.Scan(&id, &key.providerId, &key.repoId); err != nil {
			return nil, nil, paginator, errors.Err(err)
		}
		enabled[key] = id
	}

	for _, r := range rr {
		p, ok := providers[r.ProviderUserID]

		if !ok {
			continue
		}

		r.Provider = p
		r.ProviderID = p.ID

		key := enabledkey{
			providerId: r.ProviderID,
			repoId:     r.RepoID,
		}

		if id, ok := enabled[key]; ok {
			r.ID = id
			r.Enabled = true
		}
	}
	return mainProvider, rr, paginator, nil
}
