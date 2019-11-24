package oauth2

import (
	"context"
	"net/url"
	"strings"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/gitlab"
)

type Provider interface {
	Auth(c context.Context, code string, providers model.ProviderStore) error

	AuthURL() string

	AddHook(c context.Context, tok string, id int64) error

	Repos(c context.Context, tok string) ([]*model.Repo, error)

	Revoke(c context.Context, tok string) error

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
