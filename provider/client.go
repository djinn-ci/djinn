package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"

	xoauth2 "golang.org/x/oauth2"
)

type client struct {
	hookEndpoint string
	secret       string
	Endpoint     string
	Config       *xoauth2.Config
}

func (c client) Auth(ctx context.Context, code string) ([]byte, []byte, int64, error) {
	tok, err := c.Config.Exchange(ctx, code)

	if err != nil {
		return nil, nil, 0, errors.Err(err)
	}

	access, _ := crypto.Encrypt([]byte(tok.AccessToken))
	refresh, _ := crypto.Encrypt([]byte(tok.RefreshToken))

	resp, err := c.Get(tok.AccessToken, c.Endpoint+"/user")

	if err != nil {
		return nil, nil, 0, errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, 0, errors.New("unexpected http status: "+resp.Status)
	}

	u := struct{
		ID int64
	}{}

	dec := json.NewDecoder(resp.Body)
	dec.Decode(&u)
	return access, refresh, u.ID, nil
}

func (c client) AuthURL() string { return c.Config.AuthCodeURL(c.secret) }
func (c client) Secret() []byte { return []byte(c.secret) }

func (c client) do(method, tok, url string, r io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, r)

	if err != nil {
		return nil, errors.Err(err)
	}

	req.Header.Set("Authorization", "Bearer " + tok)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	cli := &http.Client{}

	resp, err := cli.Do(req)
	return resp, errors.Err(err)
}

func (c client) Get(tok, url string) (*http.Response, error) {
	resp, err := c.do("GET", tok, url, nil)
	return resp, errors.Err(err)
}

func (c client) Post(tok, url string, r io.Reader) (*http.Response, error) {
	resp, err := c.do("POST", tok, url, r)
	return resp, errors.Err(err)
}

func (c client) Delete(tok, url string) (*http.Response, error) {
	resp, err := c.do("DELETE", tok, url, nil)
	return resp, errors.Err(err)
}
