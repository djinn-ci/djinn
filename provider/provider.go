package provider

import (
	"context"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/auth/oauth2"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
)

type Provider struct {
	loaded []string
	client Client

	ID             int64
	UserID         int64
	ProviderUserID database.Null[int64]
	Name           string
	AccessToken    []byte
	RefreshToken   []byte
	Connected      bool
	MainAccount    bool
	ExpiresAt      database.Null[time.Time]
	AuthURL        string
}

var _ database.Model = (*Provider)(nil)

func (p *Provider) Primary() (string, any) { return "id", p.ID }

func (p *Provider) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":               &p.ID,
		"user_id":          &p.UserID,
		"provider_user_id": &p.ProviderUserID,
		"name":             &p.Name,
		"access_token":     &p.AccessToken,
		"refresh_token":    &p.RefreshToken,
		"connected":        &p.Connected,
		"main_account":     &p.MainAccount,
		"expires_at":       &p.ExpiresAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}

	p.loaded = r.Columns
	return nil
}

func (p *Provider) Params() database.Params {
	params := database.Params{
		"id":               database.ImmutableParam(p.ID),
		"user_id":          database.CreateOnlyParam(p.UserID),
		"provider_user_id": database.CreateUpdateParam(p.ProviderUserID),
		"name":             database.CreateUpdateParam(p.Name),
		"access_token":     database.CreateUpdateParam(p.AccessToken),
		"refresh_token":    database.CreateUpdateParam(p.RefreshToken),
		"connected":        database.CreateUpdateParam(p.Connected),
		"main_account":     database.CreateUpdateParam(p.MainAccount),
		"expires_at":       database.CreateUpdateParam(p.ExpiresAt),
	}

	if len(p.loaded) > 0 {
		params.Only(p.loaded...)
	}
	return params
}

func (*Provider) Bind(database.Model)          {}
func (*Provider) Endpoint(...string) string    { return "" }
func (*Provider) MarshalJSON() ([]byte, error) { return nil, nil }

func (p *Provider) Client() Client { return p.client }

const table = "providers"

type Store struct {
	*database.Store[*Provider]

	AuthStore auth.Store
	AESGCM    *crypto.AESGCM
	Clients   *Registry
}

func NewStore(pool *database.Pool) *database.Store[*Provider] {
	return database.NewStore[*Provider](pool, table, func() *Provider {
		return &Provider{}
	})
}

func (s *Store) Get(ctx context.Context, opts ...query.Option) (*Provider, bool, error) {
	p, ok, err := s.Store.Get(ctx, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}

	if p.AccessToken != nil {
		tok, err := s.AESGCM.Decrypt(p.AccessToken)

		if err != nil {
			return nil, false, errors.Err(err)
		}

		cli, err := s.Clients.Get(string(tok), p.Name)

		if err != nil {
			return nil, false, errors.Err(err)
		}
		p.client = cli
	}
	return p, true, nil
}

func (s *Store) Delete(ctx context.Context, pp ...*Provider) error {
	tx, err := s.Begin(ctx)

	if err != nil {
		return errors.Err(err)
	}

	defer tx.Rollback(ctx)

	p := pp[0]

	q := query.Delete(
		repoTable,
		query.Where("user_id", "=", query.Select(
			query.Columns("user_id"),
			query.From(table),
			query.Where("user_id", "=", query.Arg(p.UserID)),
			query.Where("name", "=", query.Arg(p.Name)),
		)),
	)

	if _, err := tx.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}

	q = query.Update(
		table,
		query.Set("access_token", query.Arg(nil)),
		query.Set("refresh_token", query.Arg(nil)),
		query.Set("connected", query.Arg(false)),
		query.Where("user_id", "=", query.Arg(p.UserID)),
		query.Where("name", "=", query.Arg(p.Name)),
	)

	if _, err := tx.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}

	q = query.Delete(
		table,
		query.Where("main_account", "=", query.Arg(false)),
		query.Where("user_id", "=", query.Arg(p.UserID)),
		query.Where("name", "=", query.Arg(p.Name)),
	)

	if _, err := tx.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *Store) Connected(ctx context.Context, u *auth.User) (bool, error) {
	q := query.Select(
		query.Columns("connected"),
		query.From(table),
		query.Where("user_id", "=", query.Arg(u.ID)),
		query.Where("connected", "=", query.Arg(true)),
	)

	var connected bool

	if err := s.QueryRow(ctx, q.Build(), q.Args()...).Scan(&connected); err != nil {
		if errors.Is(err, database.ErrNoRows) {
			return false, nil
		}
		return false, errors.Err(err)
	}
	return connected, nil
}

func (s *Store) Put(ctx context.Context, u *auth.User) (*auth.User, error) {
	if u.Provider == user.InternalProvider {
		return u, nil
	}

	tok := u.RawData["token"].(*oauth2.Token)

	access, err := s.AESGCM.Encrypt([]byte(tok.AccessToken))

	if err != nil {
		return nil, errors.Err(err)
	}

	refresh, err := s.AESGCM.Encrypt([]byte(tok.RefreshToken))

	if err != nil {
		return nil, errors.Err(err)
	}

	p := Provider{
		ProviderUserID: database.Null[int64]{
			Elem:  u.ID,
			Valid: true,
		},
		Name:         u.Provider,
		AccessToken:  access,
		RefreshToken: refresh,
		Connected:    true,
		MainAccount:  true,
	}

	var internal *auth.User

	if v, ok := u.RawData[user.InternalProvider]; ok {
		internal = v.(*auth.User)
	}

	if internal == nil {
		var ok bool

		internal, ok, err = user.NewStore(s.Pool).SelectOne(
			ctx,
			[]string{"id"},
			query.Where("id", "=", query.Select(
				query.Columns("user_id"),
				query.From(table),
				query.Where("provider_user_id", "=", query.Arg(u.ID)),
				query.Where("name", "=", query.Arg(u.Provider)),
				query.Where("main_account", "=", query.Arg(true)),
			)),
		)

		if err != nil {
			return nil, errors.Err(err)
		}

		if !ok {
			internal, err = s.AuthStore.Put(ctx, u)

			if err != nil {
				return nil, errors.Err(err)
			}
		}
	}

	p.UserID = internal.ID

	orig, ok, err := s.SelectOne(
		ctx,
		[]string{"id"},
		query.Where("provider_user_id", "=", query.Arg(u.ID)),
		query.Where("name", "=", query.Arg(u.Provider)),
		query.Where("main_account", "=", query.Arg(true)),
	)

	if err != nil {
		return nil, errors.Err(err)
	}

	sync := s.Create

	if ok {
		p.ID = orig.ID
		sync = s.Update
	}

	if err := sync(ctx, &p); err != nil {
		return nil, errors.Err(err)
	}

	cli, err := s.Clients.Get(tok.AccessToken, u.Provider)

	if err != nil {
		return nil, errors.Err(err)
	}

	groups, err := s.Select(
		ctx,
		[]string{"provider_user_id"},
		query.Where("user_id", "=", query.Arg(p.UserID)),
		query.Where("name", "=", query.Arg(p.Name)),
		query.Where("main_account", "=", query.Arg(false)),
	)

	if err != nil {
		return nil, errors.Err(err)
	}

	set := make(map[int64]struct{})

	for _, g := range groups {
		set[g.ProviderUserID.Elem] = struct{}{}
	}

	ids, err := cli.Groups()

	if err != nil {
		return nil, errors.Err(err)
	}

	for _, id := range ids {
		if _, ok := set[id]; ok {
			continue
		}

		tmp := p
		tmp.loaded = nil
		tmp.ProviderUserID.Elem = id
		tmp.MainAccount = false

		if err := s.Create(ctx, &tmp); err != nil {
			return nil, errors.Err(err)
		}
	}
	return internal, nil
}
