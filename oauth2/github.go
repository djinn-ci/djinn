package oauth2

import (
	"context"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	githubapi "github.com/google/go-github/github"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type GitHub struct {
	Config *oauth2.Config
}

var githubScopes = []string{
	"repo",
	"write:repo_hook",
}

func NewGitHub(id, secret string) GitHub {
	return GitHub{
		Config: &oauth2.Config{
			ClientID:     id,
			ClientSecret: secret,
			Scopes:       githubScopes,
			Endpoint:     github.Endpoint,
		},
	}
}

func (g GitHub) Auth(c context.Context, code string, providers model.ProviderStore) error {
	tok, err := g.Config.Exchange(c, code)

	if err != nil {
		return errors.Err(err)
	}

	return errors.Err(auth(c, "github", tok, providers))
}

func (g GitHub) Repos(c context.Context, tok string) ([]model.Repo, error) {
	oauthTok := &oauth2.Token{
		AccessToken: tok,
	}

	src := oauth2.StaticTokenSource(oauthTok)
	cli := githubapi.NewClient(oauth2.NewClient(c, src))

	opts := &githubapi.RepositoryListOptions{
		Sort:      "updated",
		Direction: "desc",
	}

	repos, _, err := cli.Repositories.List(c, "", opts)

	if err != nil {
		return []model.Repo{}, errors.Err(err)
	}

	rr := make([]model.Repo, 0, len(repos))

	for _, repo := range repos {
		var (
			id   int64
			name string
			href string
		)

		if repo.ID != nil {
			id = *repo.ID
		}

		if repo.FullName != nil {
			name = *repo.FullName
		}

		if repo.HTMLURL != nil {
			href = *repo.HTMLURL
		}

		r := model.Repo{
			RepoID:   id,
			Name:     name,
			Href:     href,
			Provider: &model.Provider{
				Name: "github",
			},
		}

		rr = append(rr, r)
	}

	return rr, nil
}
