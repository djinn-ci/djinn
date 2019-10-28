package oauth2

import (
	"context"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"golang.org/x/oauth2"
)

type Provider interface {
	Auth(c context.Context, code string, providers model.ProviderStore) error

	Repos(c context.Context, tok string) ([]model.Repo, error)
}

func auth(c context.Context, name string, tok *oauth2.Token, providers model.ProviderStore) error {
	access, _ := crypto.Encrypt([]byte(tok.AccessToken))
	refresh, _ := crypto.Encrypt([]byte(tok.RefreshToken))

	p, err := providers.FindByName(name)

	if err != nil {
		return errors.Err(err)
	}

	p.Name = name
	p.AccessToken = access
	p.RefreshToken = refresh
	p.ExpiresAt = tok.Expiry

	return errors.Err(providers.Update(p))
}

func NewProvider(name, id, secret string) (Provider, error) {
	switch name {
	case "github":
		return NewGitHub(id, secret), nil
	case "gitlabg":
		return NewGitLab(id, secret), nil
	default:
		return nil, errors.Err(errors.New("unknown provider '" + name + "'"))
	}
}
