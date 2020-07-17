package provider

import (
	"strings"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/oauth2"

	xoauth2 "golang.org/x/oauth2"
)

// Opts is a struct that contains common fields used across the provider
// implementations for OAuth2 authentication.
type Opts struct {
	// Host is the hostname of the server on which Thrall is running. This
	// is used when creating webhooks, and should never be localhost.
	Host string

	// Endpoint is the endpoint for the provider's API that is used when making
	// API callouts.
	Endpoint string

	// Secret is the secret string used to authenticate webhooks received from
	// the provider.
	Secret string

	ClientID     string
	ClientSecret string
}

// New returns a new oauth2.Provider for the given provider name, configuring
// the underlying client with the given Opts struct.
func New(name string, block *crypto.Block, opts Opts) (oauth2.Provider, error) {
	cli := client{
		hookEndpoint: opts.Host + "/hook/" + name,
		secret:       opts.Secret,
		block:        block,
		Endpoint:     opts.Endpoint,
		Config: &xoauth2.Config{
			ClientID:     opts.ClientID,
			ClientSecret: opts.ClientSecret,
			RedirectURL:  opts.Host + "/oauth/" + name,
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
			AuthURL:  authURL + "/login/oauth/authorize",
			TokenURL: authURL + "/login/oauth/access_token",
		}
		return GitHub{client: cli}, nil
	case "gitlab":
		if cli.Endpoint == "" {
			cli.Endpoint = gitlabURL
		}

		cli.Config.Scopes = gitlabScopes
		cli.Config.Endpoint = xoauth2.Endpoint{
			AuthURL:  cli.Endpoint + "/oauth/authorize",
			TokenURL: cli.Endpoint + "/oauth/token",
		}
		cli.Endpoint += "/api/v4"
		return GitLab{client: cli}, nil
	default:
		return nil, errors.New("unknown provider " + name)
	}
}
