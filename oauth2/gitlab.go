package oauth2

import (
	"context"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"golang.org/x/oauth2"
)

type GitLab struct {
	endpoint string

	Config *oauth2.Config
}

var gitlabScopes = []string{
	"read_repository",
	"write_repository",
}

func (g GitLab) Auth(c context.Context, code string, providers model.ProviderStore) error {
	tok, err := g.Config.Exchange(c, code)

	if err != nil {
		return errors.Err(err)
	}

	_, err = auth(c, "gitlab", tok, providers)

	return errors.Err(err)
}

func (g GitLab) AuthURL() string {
	return authURL(g.Config.Endpoint.AuthURL, g.Config.ClientID, gitlabScopes)
}

func (g GitLab) AddHook(c context.Context, tok string, id int64) error {
	return nil
}

func (g GitLab) Repos(c context.Context, tok string) ([]*model.Repo, error) {
	return []*model.Repo{}, nil
}

func (g GitLab) Revoke(c context.Context, tok string) error {
	return nil
}

func (g GitLab) Secret() []byte {
	return []byte{}
}
