package oauth2

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"

	"golang.org/x/oauth2"
)

type Provider interface {
	Auth(c context.Context, code string, providers model.ProviderStore) error

	AuthURL() string

	ToggleRepo(p *model.Provider, id int64) error

	Repos(p *model.Provider) ([]*model.Repo, error)

	Revoke(p *model.Provider) error

	Secret() []byte
}

type Client struct {
	hookEndpoint string
	secret       string
	Endpoint     string
	Config       *oauth2.Config
}

type ProviderOpts struct {
	Host         string
	Endpoint     string
	Secret       string
	ClientID     string
	ClientSecret string
}

func toggleRepo(p *model.Provider, repoId, hookId int64) error {
	repos := p.RepoStore()

	r, err := repos.Get(query.Where("repo_id", "=", repoId))

	if err != nil {
		return errors.Err(err)
	}

	create := false
	enabled := true

	if r.IsZero() {
		create = true
	} else {
		enabled = !r.Enabled
	}

	r.UserID = p.UserID
	r.ProviderID = p.ID
	r.HookID = hookId
	r.RepoID = repoId
	r.Enabled = enabled

	if create {
		return errors.Err(repos.Create(r))
	}

	return errors.Err(repos.Update(r))
}

func NewProvider(name string, opts ProviderOpts) (Provider, error) {
	cli := Client{
		hookEndpoint: opts.Host,
		secret:       opts.Secret,
		Endpoint:     opts.Endpoint,
		Config:       &oauth2.Config{
			ClientID:     opts.ClientID,
			ClientSecret: opts.ClientSecret,
		},
	}

	switch name {
	case "github":
		if cli.Endpoint == "" {
			cli.Endpoint = githubURL
		}

		cli.hookEndpoint += "/hook/github"
		cli.Config.Scopes = githubScopes
		cli.Config.Endpoint = oauth2.Endpoint{
			AuthURL:  cli.Endpoint + "/login/oauth/authorize",
			TokenURL: cli.Endpoint + "/login/oauth/access_token",
		}

		return GitHub{
			Client: cli,
		}, nil
	case "gitlab":
		if cli.Endpoint == "" {
			cli.Endpoint = gitlabURL
		}

		cli.hookEndpoint += "/hook/gitlab"
		cli.Endpoint += "/api/v4"
		cli.Config.Scopes = gitlabScopes
		cli.Config.Endpoint = oauth2.Endpoint{
			AuthURL:  cli.Endpoint + "/oauth/authorize",
			TokenURL: cli.Endpoint + "/oauth/token",
		}

		return GitLab{
			Client: cli,
		}, nil
	default:
		return nil, errors.Err(errors.New("unknown provider '" + name + "'"))
	}
}

func (c Client) auth(ctx context.Context, name, code string, providers model.ProviderStore) error {
	tok, err := c.Config.Exchange(ctx, code)

	if err != nil {
		return errors.Err(err)
	}

	access, _ := crypto.Encrypt([]byte(tok.AccessToken))
	refresh, _ := crypto.Encrypt([]byte(tok.RefreshToken))

	resp, err := c.Get(tok.AccessToken, c.Endpoint + "/user")

	if err != nil {
		return errors.Err(err)
	}

	defer resp.Body.Close()

	u := struct{
		ID int64
	}{}

	dec := json.NewDecoder(resp.Body)
	dec.Decode(&u)

	var (
		p      *model.Provider
		create bool
	)

	p, err = providers.FindByName(name)

	if err != nil {
		return errors.Err(err)
	}

	if p.IsZero() {
		p = providers.New()
		create = true
	}

	p.ProviderUserID = sql.NullInt64{
		Int64: u.ID,
		Valid: true,
	}
	p.Name = name
	p.AccessToken = access
	p.RefreshToken = refresh
	p.ExpiresAt = tok.Expiry
	p.Connected = true

	if create {
		return errors.Err(providers.Create(p))
	}

	return errors.Err(providers.Update(p))
}

func (c Client) AuthURL() string {
	return c.Config.AuthCodeURL(c.secret)
}

func (c Client) Secret() []byte {
	return []byte(c.secret)
}

func (c Client) do(method, tok, url string, r io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, r)

	if err != nil {
		return nil, errors.Err(err)
	}

	req.Header.Set("Authorization", "Bearer " + tok)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	cli := &http.Client{}

	resp, err := cli.Do(req)

	return resp, errors.Err(err)
}

func (c Client) Get(tok, url string) (*http.Response, error) {
	resp, err := c.do("GET", tok, url, nil)

	return resp, errors.Err(err)
}

func (c Client) Post(tok, url string, r io.Reader) (*http.Response, error) {
	resp, err := c.do("POST", tok, url, r)

	return resp, errors.Err(err)
}

func (c Client) Delete(tok, url string) (*http.Response, error) {
	resp, err := c.do("DELETE", tok, url, nil)

	return resp, errors.Err(err)
}
