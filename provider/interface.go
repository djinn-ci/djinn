package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/runner"

	"golang.org/x/oauth2"
)

type Client struct {
	oauth2.Config

	Host        string
	APIEndpoint string
	State       string
	Secret      string
}

type Registry struct {
	mu   sync.RWMutex
	imps map[string]Interface
}

type Interface interface {
	Auth(context.Context, url.Values) (string, string, User, error)

	AuthURL() string

	Repos(string, int64) ([]*Repo, database.Paginator, error)

	Groups(string) ([]int64, error)

	ToggleRepo(string, *Repo) error

	SetCommitStatus(string, *Repo, runner.Status, string, string) error

	VerifyRequest(io.Reader, string) ([]byte, error)
}

type Factory func(string, string, string, string, string) Interface

type User struct {
	ID       int64
	Email    string
	Login    string
	Username string
}

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrStateMismatch    = errors.New("state mismatch")

	StatusDescriptions = map[runner.Status]string{
		runner.Queued:             "Build is queued.",
		runner.Running:            "Build is running.",
		runner.Passed:             "Build has passed.",
		runner.PassedWithFailures: "Build has passed with failures.",
		runner.Failed:             "Build has failed.",
		runner.Killed:             "Build was killed.",
		runner.TimedOut:           "Build has timed out.",
	}
)

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

// GetNextAndPrev link returns the page numbers for the next and previous page
// based off the given link string. This string is expected to be the value of
// a Link header (RFC-5988, RFC-8288). By default this will return 1 for both
// values if the given header value cannot be parsed properly.
func GetNextAndPrev(link string) (int64, int64) {
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

func NewRegistry() *Registry {
	return &Registry{
		mu:   sync.RWMutex{},
		imps: make(map[string]Interface),
	}
}

func (c *Client) AuthURL() string { return c.AuthCodeURL(c.State) }

func (c *Client) Auth(ctx context.Context, v url.Values) (string, string, User, error) {
	u := User{}

	if state := v.Get("state"); state != "" {
		if state != c.State {
			return "", "", u, ErrStateMismatch
		}
	}

	tok, err := c.Exchange(ctx, v.Get("code"))

	if err != nil {
		return "", "", u, errors.Err(err)
	}

	req, err := http.NewRequest("GET", c.APIEndpoint + "/user", nil)

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

func (c *Client) do(method, tok, endpoint string, r io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.APIEndpoint + endpoint, r)

	if err != nil {
		return nil, errors.Err(err)
	}

	req.Header.Set("Authorization", "Bearer " + tok)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	cli := &http.Client{}

	resp, err := cli.Do(req)
	return resp, errors.Err(err)
}

func (c *Client) Get(tok, endpoint string) (*http.Response, error) {
	resp, err := c.do("GET", tok, endpoint, nil)
	return resp, errors.Err(err)
}

func (c *Client) Post(tok, endpoint string, r io.Reader) (*http.Response, error) {
	resp, err := c.do("POST", tok, endpoint, r)
	return resp, errors.Err(err)
}

func (c *Client) Delete(tok, endpoint string) (*http.Response, error) {
	resp, err := c.do("DELETE", tok, endpoint, nil)
	return resp, errors.Err(err)
}

func (r *Registry) All() map[string]Interface { return r.imps }

func (r *Registry) Register(name string, imp Interface) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.imps[name]; ok {
		panic("provider already registered: " + name)
	}
	r.imps[name] = imp
}

func (r *Registry) Get(name string) (Interface, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	imp, ok := r.imps[name]

	if !ok {
		return nil, errors.New("unknown provider: " + name)
	}
	return imp, nil
}
