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

// BaseClient provides a base implementation of a provider's Client. This is
// used for performing all of the HTTP requests to a provider's API.
type BaseClient struct {
	oauth2.Config

	// Host is the host of the server acting as the OAuth client for the
	// provider's OAuth server.
	Host string

	// APIEndpoint is the full URL where the provider's REST API is located.
	APIEndpoint string

	// State is the string used for verifying the final token exchange in the
	// webflow.
	State string

	// Secret is the secret that is set on each webhook created to verify the
	// request body sent by the provider.
	Secret string
}

// Registry provides a thread-safe way of registering multiple Client
// implementations against a given name, and for retrieving them at a
// later point in time.
type Registry struct {
	mu   sync.RWMutex
	imps map[string]Client
}

// Client provides a simple interface for a provider's REST API.
type Client interface {
	// Auth performs the final token exchange in the webflow. Upon success this
	// will return the access token, refresh token, and the details of the
	// User who successfully authenticated.
	Auth(context.Context, url.Values) (string, string, User, error)

	// AuthURL returns the full URL for authenticating against a provider and
	// starting the token exchange webflow.
	AuthURL() string

	// Repos returns a list of repos from the provider the user either owns or
	// has access to (via a group). The given page delineates which page of
	// repositories to get if there are multiple pages. A database.Paginator is
	// returned if there are multiple pages. This should return the repositories
	// ordered by when they were last updated.
	Repos(string, int64) ([]*Repo, database.Paginator, error)

	// Groups returns the ID of groups that a user is a member of from the
	// provider (if any/supported). This will be used to create multiple
	// providers in the database for each group, allowing for easy lookup
	// during webhook execution, should any webhook contain a group ID in place
	// of a user ID.
	Groups(string) ([]int64, error)

	// ToggleRepo will toggle the webhook for the given Repo using the given
	// string as the authentication token. This will create a webhook if it
	// doesn't exist for the Repo, otherwise it will delete it.
	ToggleRepo(string, *Repo) error

	// SetCommitStatus will set the commit status for the given commit hash to
	// the given runner.Status. This should only be called on commits that
	// belong to a merge or pull request.
	SetCommitStatus(string, *Repo, runner.Status, string, string) error

	// VerifyRequest verifies the contents of the given io.Reader against the
	// given string secret. This is typically used for verifying the contents
	// of a webhook that has been sent from said provider. Upon success it will
	// return a byte slice read from the given io.Reader.
	VerifyRequest(io.Reader, string) ([]byte, error)
}

// Factory is the type for creating a new Client for a provider. This takes the
// host of the server acting as an OAuth client, the endpoint of the provider
// for accessing the API, the secret for verifying webhooks, and finally the
// client ID, and client secret for the app we'll be granting access to.
type Factory func(string, string, string, string, string) Client

type User struct {
	ID       int64  // ID of the user from the provider.
	Email    string // Email of the user from the provider.
	Login    string // Login of the user from the provider, this is only used for GitHub.
	Username string // Username of the user from the provider.
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
		imps: make(map[string]Client),
	}
}

// AuthURL implements the Client interface.
func (c BaseClient) AuthURL() string { return c.AuthCodeURL(c.State) }

// Auth implements the Client interface.
func (c BaseClient) Auth(ctx context.Context, v url.Values) (string, string, User, error) {
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

func (c BaseClient) do(method, tok, endpoint string, r io.Reader) (*http.Response, error) {
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

// Get performs a GET request to the given endpoint using the given token
// string as the authentication token.
func (c BaseClient) Get(tok, endpoint string) (*http.Response, error) {
	resp, err := c.do("GET", tok, endpoint, nil)
	return resp, errors.Err(err)
}

// Post performs a POST request to the given endpoint using the given token
// string as the authentication token, and using the given io.Reader as the
// request body.
func (c BaseClient) Post(tok, endpoint string, r io.Reader) (*http.Response, error) {
	resp, err := c.do("POST", tok, endpoint, r)
	return resp, errors.Err(err)
}

// Delete performs a DELETE request to the given endpoint using the given token
// string as the authentication token.
func (c BaseClient) Delete(tok, endpoint string) (*http.Response, error) {
	resp, err := c.do("DELETE", tok, endpoint, nil)
	return resp, errors.Err(err)
}

// All returns all Client implementations in the current Registry.
func (r *Registry) All() map[string]Client { return r.imps }

// Register the given Client implementation against the given name. If an
// implementation is already registered this will panic.
func (r *Registry) Register(name string, imp Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.imps[name]; ok {
		panic("provider already registered: " + name)
	}
	r.imps[name] = imp
}

// Get returns a Client implementation for the given name. If no implementation
// is found then nil is returned along with an error.
func (r *Registry) Get(name string) (Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	imp, ok := r.imps[name]

	if !ok {
		return nil, errors.New("unknown provider: " + name)
	}
	return imp, nil
}
