package oauth2

import (
	"context"
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/google/go-github/github"

	"golang.org/x/oauth2"
)

type GitHub struct {
	endpoint string
	secret   string

	Config *oauth2.Config
}

var githubScopes = []string{
	"repo",
	"admin:repo_hook",
}

func githubClient(c context.Context, tok string) *github.Client {
	oauthTok := &oauth2.Token{
		AccessToken: tok,
	}

	src := oauth2.StaticTokenSource(oauthTok)

	return github.NewClient(oauth2.NewClient(c, src))
}

func (g GitHub) Auth(c context.Context, code string, providers model.ProviderStore) error {
	tok, err := g.Config.Exchange(c, code)

	if err != nil {
		return errors.Err(err)
	}

	return errors.Err(auth(c, "github", tok, providers))
}

func (g GitHub) AuthURL() string {
	return authURL(g.Config.Endpoint.AuthURL, g.Config.ClientID, githubScopes)
}

func (g GitHub) AddHook(c context.Context, tok string, id int64) error {
	cli := githubClient(c, tok)

	repo, resp, err := cli.Repositories.GetByID(c, id)

	if err != nil {
		return errors.Err(err)
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Err(errors.New("failed to get repository: " + resp.Status))
	}

	h := &github.Hook{
		Config: map[string]interface{}{
			"url":          g.endpoint,
			"secret":       g.secret,
			"content_type": "json",
			"insecure_ssl": 0,
		},
		Events: []string{
			"push",
			"pull_request",
		},
	}

	_, resp, err = cli.Repositories.CreateHook(c, *repo.Owner.Login, *repo.Name, h)

	if err != nil {
		return errors.Err(err)
	}

	if resp.StatusCode != http.StatusCreated {
		return errors.Err(errors.New("failed to get repository: " + resp.Status))
	}

	return nil
}

func (g GitHub) Repos(c context.Context, tok string) ([]*model.Repo, error) {
	cli := githubClient(c, tok)

	opts := &github.RepositoryListOptions{
		Sort:      "updated",
		Direction: "desc",
	}

	repos, _, err := cli.Repositories.List(c, "", opts)

	if err != nil {
		return []*model.Repo{}, errors.Err(err)
	}

	rr := make([]*model.Repo, 0, len(repos))

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

		r := &model.Repo{
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

func (g GitHub) Revoke(c context.Context, tok string) error {
	transport := &github.BasicAuthTransport{
		Username: g.Config.ClientID,
		Password: g.Config.ClientSecret,
	}

	cli := github.NewClient(&http.Client{
		Transport: transport,
	})

	resp, err := cli.Authorizations.Revoke(c, g.Config.ClientID, tok)

	if err != nil {
		return errors.Err(err)
	}

	if resp.Response.StatusCode != http.StatusNoContent {
		return errors.Err(errors.New("unexpected response from api: " + resp.Response.Status))
	}

	return nil
}
