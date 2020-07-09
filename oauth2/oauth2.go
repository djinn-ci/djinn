// Package oauth2 provides model implementations for and functionality for the
// OAuth2 server. This also provides an interface for implementations of OAuth2
// clients to allow for authentication against the server.
package oauth2

import "context"

// Provider is an interface that can be implemented to support a Git provider
// that we wish to use to Auth against, and add hooks to for build submission.
type Provider interface {
	// Auth should perform the authentication flow, and token exchange of an
	// OAuth2 client. This should return the access token, refresh token, and
	// user ID of the user from the provider itself.
	Auth(context.Context, string) ([]byte, []byte, User, error)

	// AuthURL should return the URL that we should take the user to in order
	// to begin the web authentication flow.
	AuthURL() string

	// ToggleRepo should toggle the repository hook for the given repository
	// ID. The first byte slice is the access token that should be used for
	// performing auth against the remote provider, and the callback should
	// be used to determine if the repo hook should be added or removed. On
	// a successful enabling of a hook the ID for that hook should be returned.
	// On a successful disabling of a hook, 0 should be returned.
	ToggleRepo([]byte, int64, func(int64) (int64, bool, error)) (int64, error)

	// Repos should return a paginate list of repos for the given page.
	Repos([]byte, int64) (Repos, error)

	// Secret should return the secret used to verify the webhooks that are
	// received from the provider.
	Secret() []byte
}

// User represents the bare minimum information about the user from the
// provider. This is used for creating a user account on the fly if one does
// not currently exist.
type User struct {
	ID       int64
	Email    string
	Login    string
	Username string
}

// Repos is a simple struct that represents a paginated response from a
// provider of repositories.
type Repos struct {
	Next  int64
	Prev  int64
	Items []Repo
}

// Repo is a simple struct that represents the bare minimum information we
// need to display the repositories available to a user from a provider.
type Repo struct {
	ID   int64
	Name string

	// Href is the link to the repository itself on the provider.
	Href string
}
