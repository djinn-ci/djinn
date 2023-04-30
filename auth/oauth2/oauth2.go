package oauth2

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"djinn-ci.com/auth"

	"golang.org/x/oauth2"
)

type Token = oauth2.Token

var ErrStateMismatch = errors.New("auth/oauth2: state mismatch")

type GetUserFunc func(cli *http.Client, tok *Token) (*auth.User, error)

type Config struct {
	oauth2.Config

	State   string
	GetUser GetUserFunc
}

type Endpoint struct {
	OAuth string
	API   string
}

func GetUser(provider, endpoint string) GetUserFunc {
	return func(cli *http.Client, tok *Token) (*auth.User, error) {
		req, err := http.NewRequest("GET", endpoint, nil)

		if err != nil {
			return nil, err
		}

		req.Header.Set("Accept", "application/json; charset=utf-8")
		req.Header.Set("Authorization", "Bearer "+tok.AccessToken)

		resp, err := cli.Do(req)

		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("auth/oauth2: GET %s unexpected status %s", endpoint, resp.Status)
		}

		var (
			raw map[string]any = make(map[string]any)
			buf bytes.Buffer
			u   auth.User
		)

		json.NewDecoder(io.TeeReader(resp.Body, &buf)).Decode(&u)
		json.NewDecoder(&buf).Decode(&raw)

		u.Provider = provider
		u.RawData = raw
		u.RawData["token"] = tok

		return &u, nil
	}
}

type Authenticator struct {
	cli  *http.Client
	cfg  *Config
	name string
}

func New(name string, cli *http.Client, cfg *Config) *Authenticator {
	if cli == nil {
		cli = http.DefaultClient
		cli.Timeout = time.Minute
	}

	return &Authenticator{
		cli:  cli,
		cfg:  cfg,
		name: name,
	}
}

func (a *Authenticator) AuthURL() string {
	return a.cfg.AuthCodeURL(a.cfg.State)
}

func (a *Authenticator) Auth(r *http.Request) (*auth.User, error) {
	q := r.URL.Query()

	if state := q.Get("state"); state != "" {
		if state != a.cfg.State {
			return nil, ErrStateMismatch
		}
	}

	tok, err := a.cfg.Exchange(r.Context(), q.Get("code"))

	if err != nil {
		return nil, err
	}

	u, err := a.cfg.GetUser(a.cli, tok)

	if err != nil {
		return nil, err
	}
	return u, nil
}
