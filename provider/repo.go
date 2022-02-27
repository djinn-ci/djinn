package provider

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"

	"github.com/go-redis/redis"
)

type Repo struct {
	ID             int64
	UserID         int64
	ProviderID     int64 `gob:"-"`
	ProviderUserID int64
	HookID         sql.NullInt64
	RepoID         int64
	ProviderName   string
	Enabled        bool
	Name           string
	Href           string

	User     *user.User `gob:"-"`
	Provider *Provider  `gob:"-"`
}

var _ database.Model = (*Repo)(nil)

func (r *Repo) Dest() []interface{} {
	return []interface{}{
		&r.ID,
		&r.UserID,
		&r.ProviderID,
		&r.HookID,
		&r.RepoID,
		&r.ProviderName,
		&r.Enabled,
		&r.Name,
		&r.Href,
	}
}

func (r *Repo) Bind(m database.Model) {
	switch v := m.(type) {
	case *user.User:
		if r.UserID == v.ID {
			r.User = v
		}
	case *Provider:
		if r.ProviderID == v.ID {
			r.Provider = v
		}
	}
}
func (*Repo) JSON(_ string) map[string]interface{} { return nil }

func (r *Repo) Endpoint(uri ...string) string {
	if len(uri) > 0 {
		return "/repos/" + strconv.FormatInt(r.ID, 10) + "/" + strings.Join(uri, "/")
	}
	return "/repos/" + strconv.FormatInt(r.ID, 10)
}

func (r *Repo) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":            r.ID,
		"user_id":       r.UserID,
		"provider_id":   r.ProviderID,
		"hook_id":       r.HookID,
		"repo_id":       r.RepoID,
		"provider_name": r.ProviderName,
		"enabled":       r.Enabled,
		"name":          r.Name,
		"href":          r.Href,
	}
}

type RepoStore struct {
	database.Pool
}

var repoTable = "provider_repos"

func (s RepoStore) Touch(r *Repo) error {
	if r.ID == 0 {
		q := query.Insert(
			repoTable,
			query.Columns("user_id", "provider_id", "hook_id", "repo_id", "provider_name", "enabled", "name", "href"),
			query.Values(r.UserID, r.ProviderID, r.HookID, r.RepoID, r.ProviderName, r.Enabled, r.Name, r.Href),
		)

		if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
			return errors.Err(err)
		}
		return nil
	}

	q := query.Update(
		repoTable,
		query.Set("hook_id", query.Arg(r.HookID)),
		query.Set("enabled", query.Arg(r.Enabled)),
		query.Where("id", "=", query.Arg(r.ID)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s RepoStore) Delete(id int64) error {
	q := query.Delete(repoTable, query.Where("id", "=", query.Arg(id)))

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s RepoStore) Get(opts ...query.Option) (*Repo, bool, error) {
	var r Repo

	ok, err := s.Pool.Get(repoTable, &r, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &r, ok, nil
}

func (s RepoStore) All(opts ...query.Option) ([]*Repo, error) {
	rr := make([]*Repo, 0)

	new := func() database.Model {
		r := &Repo{}
		rr = append(rr, r)
		return r
	}

	if err := s.Pool.All(repoTable, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return rr, nil
}

type RepoCache struct {
	redis *redis.Client
}

var repoCacheKey = "repos-%s-%v-%v"

func NewRepoCache(redis *redis.Client) RepoCache {
	return RepoCache{
		redis: redis,
	}
}

func (c RepoCache) key(p *Provider, page int64) string {
	return fmt.Sprintf(repoCacheKey, p.Name, p.UserID, page)
}

func (c RepoCache) Put(p *Provider, rr []*Repo, paginator database.Paginator) error {
	var buf bytes.Buffer

	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(rr); err != nil {
		return errors.Err(err)
	}

	if err := enc.Encode(paginator); err != nil {
		return errors.Err(err)
	}

	if _, err := c.redis.Set(c.key(p, paginator.Page), buf.String(), time.Hour).Result(); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (c RepoCache) Get(p *Provider, page int64) ([]*Repo, database.Paginator, error) {
	var paginator database.Paginator

	rr := make([]*Repo, 0)

	s, err := c.redis.Get(c.key(p, page)).Result()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, paginator, nil
		}
		return nil, paginator, errors.Err(err)
	}

	dec := gob.NewDecoder(strings.NewReader(s))

	if err := dec.Decode(&rr); err != nil {
		return nil, paginator, errors.Err(err)
	}

	if err := dec.Decode(&paginator); err != nil {
		return nil, paginator, errors.Err(err)
	}
	return rr, paginator, nil
}

func (c RepoCache) Purge(p *Provider) error {
	page := int64(1)

	for {
		key := c.key(p, page)

		n, err := c.redis.Del(key).Result()

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
