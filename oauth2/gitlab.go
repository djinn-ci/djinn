package oauth2

import (
	"context"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/gitlab"
)

type GitLab struct {
	Config *oauth2.Config
}

var gitlabScopes = []string{
	"read_repository",
	"write_repository",
}

func NewGitLab(id, secret string) GitLab {
	return GitLab{
		Config: &oauth2.Config{
			ClientID:     id,
			ClientSecret: secret,
			Scopes:       gitlabScopes,
			Endpoint:     gitlab.Endpoint,
		},
	}
}

func (g GitLab) Auth(c context.Context, code string, providers model.ProviderStore) error {
	tok, err := g.Config.Exchange(c, code)

	if err != nil {
		return errors.Err(err)
	}

	return errors.Err(auth(c, "gitlab", tok, providers))
}

func (g GitLab) AuthURL() string {
	return authURL(g.Config.Endpoint.AuthURL, g.Config.ClientID, gitlabScopes)
}

func (g GitLab) Repos(c context.Context, tok string) ([]model.Repo, error) {
	return []model.Repo{}, nil
}
