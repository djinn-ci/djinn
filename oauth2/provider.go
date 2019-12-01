package oauth2

import (
	"context"
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

func auth(c context.Context, name string, tok *oauth2.Token, providers model.ProviderStore) (*model.Provider, error) {
	access, _ := crypto.Encrypt([]byte(tok.AccessToken))
	refresh, _ := crypto.Encrypt([]byte(tok.RefreshToken))

	p, err := providers.FindByName(name)

	if err != nil {
		return p, errors.Err(err)
	}

	p.Name = name
	p.AccessToken = access
	p.RefreshToken = refresh
	p.ExpiresAt = tok.Expiry
	p.Connected = true

	return p, nil
}

func authURL(rawurl, id string, scopes []string) string {
	url, _ := url.Parse(rawurl)

	q := url.Query()
	q.Add("client_id", id)
	q.Add("scope", strings.Join(scopes, " "))

	url.RawQuery = q.Encode()

	return url.String()
}

func httpGet(tok, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, errors.Err(err)
	}

	req.Header.Set("Authorization", tok)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	cli := &http.Client{}

	resp, err := cli.Do(req)

	return resp, errors.Err(err)
}

func httpPost(tok, url string, r io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, r)

	if err != nil {
		return nil, errors.Err(err)
	}

	req.Header.Set("Authorization", tok)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	cli := &http.Client{}

	resp, err := cli.Do(req)

	return resp, errors.Err(err)
}

func httpDelete(tok, url string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", url, nil)

	if err != nil {
		return nil, errors.Err(err)
	}

	req.Header.Set("Authorization", tok)

	cli := &http.Client{}

	resp, err := cli.Do(req)

	return resp, errors.Err(err)
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

func NewProvider(name, clientId, clientSecret, host, secret string) (Provider, error) {
	cfg := &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
	}

	switch name {
	case "github":
		cfg.Scopes = githubScopes
		cfg.Endpoint = github.Endpoint

		return GitHub{
			endpoint: host + "/hook/github",
			secret:   secret,
			Config:   cfg,
		}, nil
	case "gitlab":
		cfg.Scopes = gitlabScopes
		cfg.Endpoint = gitlab.Endpoint

		return GitLab{
			endpoint: host + "/hook/gitlab",
			Config:   cfg,
		}, nil
	default:
		return nil, errors.Err(errors.New("unknown provider '" + name + "'"))
	}
}
