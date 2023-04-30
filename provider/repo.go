package provider

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"

	"github.com/andrewpillar/query"

	"github.com/go-redis/redis"
)

type Repo struct {
	loaded []string

	ID             int64
	UserID         int64
	ProviderID     int64 `gob:"-"`
	ProviderUserID int64
	HookID         database.Null[int64]
	RepoID         int64
	ProviderName   string
	Enabled        bool
	Name           string
	Href           string

	Provider *Provider `gob:"-"`
}

var _ database.Model = (*Repo)(nil)

func (r *Repo) Primary() (string, any) { return "id", r.ID }

func (r *Repo) Scan(row *database.Row) error {
	valtab := map[string]any{
		"id":            &r.ID,
		"user_id":       &r.UserID,
		"provider_id":   &r.ProviderID,
		"hook_id":       &r.HookID,
		"repo_id":       &r.RepoID,
		"provider_name": &r.ProviderName,
		"enabled":       &r.Enabled,
		"name":          &r.Name,
		"href":          &r.Href,
	}

	if err := database.Scan(row, valtab); err != nil {
		return errors.Err(err)
	}

	r.loaded = row.Columns
	return nil
}

func (r *Repo) Params() database.Params {
	params := database.Params{
		"id":            database.ImmutableParam(r.ID),
		"user_id":       database.CreateOnlyParam(r.UserID),
		"provider_id":   database.CreateOnlyParam(r.ProviderID),
		"hook_id":       database.CreateUpdateParam(r.HookID),
		"repo_id":       database.CreateUpdateParam(r.RepoID),
		"provider_name": database.CreateUpdateParam(r.ProviderName),
		"enabled":       database.CreateUpdateParam(r.Enabled),
		"name":          database.CreateUpdateParam(r.Name),
		"href":          database.CreateUpdateParam(r.Href),
	}

	if len(r.loaded) > 0 {
		params.Only(r.loaded...)
	}
	return params
}

func (r *Repo) Bind(m database.Model) {
	if v, ok := m.(*Provider); ok {
		if r.ProviderID == v.ID {
			r.Provider = v
		}
	}
}

func (*Repo) MarshalJSON() ([]byte, error) { return nil, nil }

func (rp *Repo) Endpoint(elems ...string) string {
	if len(elems) > 0 {
		return "/repos/" + strconv.FormatInt(rp.ID, 10) + "/" + strings.Join(elems, "/")
	}
	return "/repos/" + strconv.FormatInt(rp.ID, 10)
}

type RepoStore struct {
	*database.Store[*Repo]

	Cache *redis.Client
}

const repoTable = "provider_repos"

func NewRepoStore(pool *database.Pool) *database.Store[*Repo] {
	return database.NewStore[*Repo](pool, repoTable, func() *Repo {
		return &Repo{}
	})
}

func (s *RepoStore) cacheKey(p *Provider, page int) string {
	return fmt.Sprintf("repos-%s-%v-%v", p.Name, p.UserID, page)
}

func (s *RepoStore) getCached(p *Provider, page int) (*database.Paginator[*Repo], error) {
	str, err := s.Cache.Get(s.cacheKey(p, page)).Result()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return &database.Paginator[*Repo]{}, nil
		}
		return nil, errors.Err(err)
	}

	var repos database.Paginator[*Repo]

	if err := gob.NewDecoder(strings.NewReader(str)).Decode(&repos); err != nil {
		return nil, errors.Err(err)
	}
	return &repos, nil
}

func (s *RepoStore) cache(p *Provider, repos *database.Paginator[*Repo]) error {
	var buf bytes.Buffer

	if err := gob.NewEncoder(&buf).Encode(repos); err != nil {
		return errors.Err(err)
	}

	if _, err := s.Cache.Set(s.cacheKey(p, repos.Page()), buf.String(), time.Hour).Result(); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *RepoStore) Reload(ctx context.Context, p *Provider, page int) (*database.Paginator[*Repo], error) {
	if !p.Connected {
		return nil, nil
	}

	repos, err := p.Client().Repos(page)

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := s.cache(p, repos); err != nil {
		return nil, errors.Err(err)
	}
	return repos, nil
}

func (s *RepoStore) Load(ctx context.Context, p *Provider, page int) (*database.Paginator[*Repo], error) {
	if !p.Connected {
		return nil, nil
	}

	rr, err := s.All(
		ctx,
		query.Where("provider_id", "=", query.Arg(p.ID)),
		query.Where("enabled", "=", query.Arg(true)),
	)

	if err != nil {
		return nil, errors.Err(err)
	}

	type key struct {
		providerId, repoId int64
	}

	enabled := make(map[key]int64)

	for _, r := range rr {
		key := key{
			providerId: r.ProviderID,
			repoId:     r.RepoID,
		}
		enabled[key] = r.ID
	}

	cached, err := s.getCached(p, page)

	if err != nil {
		return nil, errors.Err(err)
	}

	if len(cached.Items) == 0 {
		cached, err = p.Client().Repos(page)

		if err != nil {
			return nil, errors.Err(err)
		}

		if len(cached.Items) > 0 {
			cached.Set(page)

			if err := s.cache(p, cached); err != nil {
				return nil, errors.Err(err)
			}
		}
	}

	cached.Set(page)

	for _, r := range cached.Items {
		key := key{
			providerId: r.ProviderID,
			repoId:     r.RepoID,
		}

		r.Provider = p

		if id, ok := enabled[key]; ok {
			r.ID = id
			r.Enabled = true
		}
	}
	return cached, nil
}

func (s *RepoStore) Touch(ctx context.Context, r *Repo) error {
	if r.ID == 0 {
		if err := s.Store.Create(ctx, r); err != nil {
			return errors.Err(err)
		}
		return nil
	}

	if err := s.Store.Update(ctx, r); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *RepoStore) Purge(p *Provider) error {
	page := 1

	for {
		key := s.cacheKey(p, page)

		n, err := s.Cache.Del(key).Result()

		if err != nil {
			return errors.Err(err)
		}

		if n == 0 {
			break
		}
		page++
	}
	return nil
}
