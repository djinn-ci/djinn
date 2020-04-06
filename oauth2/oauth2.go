package oauth2

import "context"

type Provider interface {
	Auth(context.Context, string) ([]byte, []byte, int64, error)

	AuthURL() string

	ToggleRepo([]byte, int64, func(int64) (int64, bool, error)) (int64, error)

	Repos([]byte) ([]Repo, error)

	Revoke([]byte) error

	Secret() []byte
}

type Repo struct {
	ID   int64
	Name string
	Href string
}
