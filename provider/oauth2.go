package provider

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"

	"golang.org/x/oauth2"
)

type Config struct {
	oauth2.Config

	endpoint string
	Secret   string
}

type Client struct {
	host     string
	endpoint string
	block    *crypto.Block
	Secret   string
}

type Registry struct {
	l       int
	mu      sync.RWMutex
	configs map[string]Config
	clients map[string]Client
}

type User struct {
	ID       int64
	Email    string
	Login    string
	Username string
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

		for i := 0; i < len(part)-1; i++ {
			b := part[i]

			// We hit the raw URL, so set the dst to point to the raw slice.
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
				// Make sure we're in bounds of what we're scanning when getting
				// the rel attribute of the value.
				if i+6 > len(part)-1 {
					continue
				}

				// Check to see if the next part is the rel attribute, if so
				// then we skip ahead to the actual value of the attribute.
				if part[i+2:i+5] == "rel" {
					scan = true
					dst = &rel
					i += 6
					continue
				}
			}
			// Put whatever we have into the destination byte slice pointed to
			// via dst. This will either be the raw URL or the value of rel.
			if scan {
				(*dst) = append((*dst), b)
			}
		}
		m[string(rel)] = append(m[string(rel)], string(raw))
	}
	return m
}

func NewConfig(name, host, endpoint, clientId, clientSecret string) (Config, error) {
	cfg := Config{
		Config:   oauth2.Config{
			ClientID:     clientId,
			ClientSecret: clientSecret,
			RedirectURL:  host + "/oauth/" + name,
		},
		endpoint: endpoint,
	}

	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		return cfg, errors.Err(err)
	}

	cfg.Secret = hex.EncodeToString(b)

	switch name {
	case "github":
		if endpoint == "" {
			cfg.endpoint = githubURL
		}

		url := strings.Replace(cfg.endpoint, "api.", "", 1)

		cfg.Config.Scopes = githubScopes
		cfg.Config.Endpoint = oauth2.Endpoint{
			AuthURL:  url + "/login/oauth/authorize",
			TokenURL: url + "/login/oauth/access_token",
		}
		return cfg, nil
	case "gitlab":
		if endpoint == "" {
			cfg.endpoint = gitlabURL
		}

		url := strings.Replace(cfg.endpoint, "/api/v4", "", 1)

		cfg.Config.Scopes = gitlabScopes
		cfg.Config.Endpoint = oauth2.Endpoint{
			AuthURL:  url + "/oauth/authorize",
			TokenURL: url + "/oauth/token",
		}
		return cfg, nil
	default:
		return Config{}, errors.New("unknown provider: " + name)
	}
}

func NewClient(name, secret, host, endpoint string, block *crypto.Block) (Client, error) {
	cli := Client{
		host:     host,
		endpoint: endpoint,
		Secret:   secret,
		block:    block,
	}

	switch name {
	case "github":
		if cli.endpoint == "" {
			cli.endpoint = githubURL
		}
		return cli, nil
	case "gitlab":
		if cli.endpoint == "" {
			cli.endpoint = gitlabURL
		}
		return cli, nil
	default:
		return Client{}, errors.New("unknown provider: " + name)
	}
}

func NewRegistry() *Registry {
	return &Registry{
		mu:        sync.RWMutex{},
		configs:   make(map[string]Config),
		clients:   make(map[string]Client),
	}
}

func (c Config) Auth(ctx context.Context, code string) (string, string, User, error) {
	u := User{}

	tok, err := c.Exchange(ctx, code)

	if err != nil {
		return "", "", u, errors.Err(err)
	}

	req, err := http.NewRequest("GET", c.endpoint + "/user", nil)

	if err != nil {
		return "", "", u, errors.Err(err)
	}

	req.Header.Set("Authorization", "Bearer " + tok.AccessToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	cli := http.Client{}

	resp, err := cli.Do(req)

	if err != nil {
		return "", "", u, errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", u, errors.New("unexpected http status: " + resp.Status)
	}

	json.NewDecoder(resp.Body).Decode(&u)
	return tok.AccessToken, tok.RefreshToken, u, nil
}

func (c Client) do(method, tok, endpoint string, r io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.endpoint + endpoint, r)

	if err != nil {
		return nil, errors.Err(err)
	}

	req.Header.Set("Authorization", "Bearer " + tok)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	cli := &http.Client{}

	resp, err := cli.Do(req)
	return resp, errors.Err(err)
}

func (c Client) Get(tok, endpoint string) (*http.Response, error) {
	resp, err := c.do("GET", tok, endpoint, nil)
	return resp, errors.Err(err)
}

func (c Client) Post(tok, endpoint string, r io.Reader) (*http.Response, error) {
	resp, err := c.do("POST", tok, endpoint, r)
	return resp, errors.Err(err)
}

func (c Client) Delete(tok, endpoint string) (*http.Response, error) {
	resp, err := c.do("DELETE", tok, endpoint, nil)
	return resp, errors.Err(err)
}

func (r *Registry) Register(name string, cfg Config, cli Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok1 := r.configs[name]
	_, ok2 := r.clients[name]

	if ok1 && ok2 {
		panic("provider already registered: " + name)
	}
	r.l++
	r.configs[name] = cfg
	r.clients[name] = cli
}

func (r *Registry) All() (map[string]Config, map[string]Client) { return r.configs, r.clients }

func (r *Registry) Get(name string) (Config, Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfg, ok1 := r.configs[name]
	cli, ok2 := r.clients[name]

	if !ok1 && !ok2 {
		return cfg, cli, errors.New("unknown provider: " + name)
	}
	return cfg, cli, nil
}
