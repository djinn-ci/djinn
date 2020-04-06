package provider

import (
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/oauth2"

	xoauth2 "golang.org/x/oauth2"
)

type Opts struct {
	Host         string
	Endpoint     string
	Secret       string
	ClientID     string
	ClientSecret string
}

func New(name string, opts Opts) (oauth2.Provider, error) {
	cli := client{
		hookEndpoint: opts.Host+"/hook/"+name,
		secret:       opts.Secret,
		Endpoint:     opts.Endpoint,
		Config:       &xoauth2.Config{
			ClientID:     opts.ClientID,
			ClientSecret: opts.ClientSecret,
			RedirectURL:  opts.Host+"/oauth/"+name,
		},
	}

	switch name {
	case "github":
		if cli.Endpoint == "" {
			cli.Endpoint = githubURL
		}

		authURL := strings.Replace(cli.Endpoint, "api.", "", 1)

		cli.Config.Scopes = githubScopes
		cli.Config.Endpoint = xoauth2.Endpoint{
			AuthURL:  authURL+"/login/oauth/authorize",
			TokenURL: authURL+"/login/oauth/access_token",
		}
		return GitHub{client: cli}, nil
	case "gitlab":
		if cli.Endpoint == "" {
			cli.Endpoint = gitlabURL
		}

		cli.Config.Scopes = gitlabScopes
		cli.Config.Endpoint = xoauth2.Endpoint{
			AuthURL:  cli.Endpoint+"/oauth/authorize",
			TokenURL: cli.Endpoint+"/oauth/token",
		}
		cli.Endpoint += "/api/v4"
		return GitLab{client: cli}, nil
	default:
		return nil, errors.New("unknown provider "+name)
	}
}
