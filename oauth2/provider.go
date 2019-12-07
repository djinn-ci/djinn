package oauth2

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/gitlab"
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
	APIEndpoint string
	Config      *oauth2.Config
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

func NewProvider(name, clientId, clientSecret, host, secret, endpoint string) (Provider, error) {
	cli := Client{
		Config: &oauth2.Config{
			ClientID:     clientId,
			ClientSecret: clientSecret,
		},
	}

	switch name {
	case "github":
		cli.APIEndpoint = githubURL
		cli.Config.Scopes = githubScopes
		cli.Config.Endpoint = github.Endpoint

		return GitHub{
			Client:       cli,
			hookEndpoint: host + "/hook/github",
			secret:       secret,
		}, nil
	case "gitlab":
		if endpoint == "" {
			endpoint = gitlabURL
		}

		cli.APIEndpoint = endpoint + "/api/v4"
		cli.Config.Scopes = gitlabScopes
		cli.Config.Endpoint = gitlab.Endpoint

		if endpoint != "" {
			cli.Config.Endpoint.AuthURL = endpoint + "/oauth/authorize"
			cli.Config.Endpoint.TokenURL = endpoint + "/oauth/token"
		}

		return GitLab{
			Client:       cli,
			hookEndpoint: host + "/hook/gitlab",
			secret:       secret,
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

	resp, err := c.Get(tok.AccessToken, c.APIEndpoint + "/user")

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
	u, _ := url.Parse(c.Config.Endpoint.AuthURL)

	q := u.Query()
	q.Add("client_id", c.Config.ClientID)
	q.Add("scope", strings.Join(c.Config.Scopes, " "))

	u.RawQuery = q.Encode()

	return u.String()
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
