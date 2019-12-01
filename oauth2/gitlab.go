package oauth2

import (
	"context"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
)

type GitLab struct {
	Client

	hookEndpoint string
	secret       string
}

var (
	gitlabScopes = []string{
		"read_repository",
		"write_repository",
	}

	gitlabURL = "https://gitlab.com"
)

func (g GitLab) Auth(c context.Context, code string, providers model.ProviderStore) error {
	return errors.Err(g.auth(c, "gitlab", code, providers))
}

func (g GitLab) ToggleRepo(p *model.Provider, id int64) error {
	return nil
}

func (g GitLab) Repos(p *model.Provider) ([]*model.Repo, error) {
	return []*model.Repo{}, nil
}

func (g GitLab) Revoke(p *model.Provider) error {
	return nil
}

func (g GitLab) Secret() []byte {
	return []byte{}
}
