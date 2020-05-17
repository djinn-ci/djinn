package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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

// getNextAndPrev link returns the page numbers for the next and previous page
// based off the given link string. This string is expected to be the value of
// a Link header (RFC-5988, RFC-8288). By default this will return 1 for both
// values if the given header value cannot be parsed properly.
func getNextAndPrev(link string) (int64, int64) {
	var (
		next int64 = 1
		prev int64 = 1
	)

	if link == "" {
		return next, prev
	}

	links := parseLink(link)

	if urls, ok := links["next"]; ok {
		url, _ := url.Parse(urls[0])
		page, err := strconv.ParseInt(url.Query().Get("page"), 10, 64)

		if err != nil {
			page = 1
		}
		next = page
	}

	if urls, ok := links["prev"]; ok {
		url, _ := url.Parse(urls[0])
		page, err := strconv.ParseInt(url.Query().Get("page"), 10, 64)

		if err != nil {
			page = 1
		}
		prev = page
	}
	return next, prev
}

// parseLink parses the given string value as the value of the Link header
// (RFC-5988, RFC-8288). The rel for each of these links is used as the key,
// and if duplicate rels are provided then the values are simply appended to
// the respective string slice in the returned map.
func parseLink(link string) map[string][]string {
	m := make(map[string][]string)
	parts := strings.Split(link, ",")

	for _, part := range parts {
		var (
			raw  []byte
			rel  []byte
			dst  *[]byte
			scan bool
		)

		for i := 0; i < len(part) - 1; i++ {
			b := part[i]

			if b == '<' {
				scan = true
				dst = &raw
				continue
			}
			if b == '>' {
				scan = false
				continue
			}
			if b == ';' {
				if i + 6 > len(part) - 1 {
					continue
				}
				if part[i+2:i+5] == "rel" {
					scan = true
					dst = &rel
					i += 6
					continue
				}
			}
			if scan {
				(*dst) = append((*dst), b)
			}
		}
		m[string(rel)] = append(m[string(rel)], string(raw))
	}
	return m
}

// Auth performs the final stage of web flow authentication for an OAuth2
// client. This will return the access, and refresh token. The returned user ID
// is retrieved via a callout to the APIs /user endpoint, this of course
// assumes that the API provides said endpoint, and returns a JSON response
// contains an integer for the user's ID.
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

// AuthURL returns the full redirect URL to begin the web flow auth.
func (c client) AuthURL() string { return c.Config.AuthCodeURL(c.secret) }

// Secret returns the secret used for verifying webhooks triggered by the
// provider.
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

// Get performs an HTTP GET request to the given URL. This sets the
// Authorization header on the outgoing request to Bearer with the given token
// value. This set's the Content-Type header to
// "application/json; charset=utf-8".
func (c client) Get(tok, url string) (*http.Response, error) {
	resp, err := c.do("GET", tok, url, nil)
	return resp, errors.Err(err)
}

// Post performs an HTTP POST request to the given URL, using the given
// io.Reader for the request's body. This sets the Authorization header on the
// outgoing request to Bearer with the given token value. This set's the
// Content-Type header to "application/json; charset=utf-8".
func (c client) Post(tok, url string, r io.Reader) (*http.Response, error) {
	resp, err := c.do("POST", tok, url, r)
	return resp, errors.Err(err)
}

// Get performs an HTTP DELETE request to the given URL. This sets the
// Authorization header on the outgoing request to Bearer with the given token
// value. This set's the Content-Type header to
// "application/json; charset=utf-8".
func (c client) Delete(tok, url string) (*http.Response, error) {
	resp, err := c.do("DELETE", tok, url, nil)
	return resp, errors.Err(err)
}
