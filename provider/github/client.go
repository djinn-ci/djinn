// Package github provides an implementation of the provider.Client interface
// for the GitHub REST API.
package github

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/provider"
	"djinn-ci.com/runner"

	"golang.org/x/oauth2"
)

type Client struct {
	provider.BaseClient
}

type Error struct {
	Message string              // Message is the original message from a failed request.
	Errors  []map[string]string // Errors is the list of errors that occurred from a failed request.
}

type User struct {
	ID    int64
	Login string
}

type Repo struct {
	ID          int64
	Owner       User
	FullName    string `json:"full_name"`
	HTMLURL     string `json:"html_url"`
	ContentsURL string `json:"contents_url"`
}

type PushEvent struct {
	Ref        string
	Repo       Repo `json:"repository"`
	HeadCommit struct {
		ID      string
		URL     string
		Message string
		Author  map[string]string
	} `json:"head_commit"`
}

type PullRequestEvent struct {
	Action      string
	Number      int64
	PullRequest struct {
		Title   string
		User    User
		HTMLURL string `json:"html_url"`
		Head    struct {
			Sha  string
			User User
			Repo Repo
		}
		Base struct {
			Ref  string
			User User
			Repo Repo
		}
	} `json:"pull_request"`
}

var (
	_ provider.Client = (*Client)(nil)

	signatureLength = 45

	states = map[runner.Status]string{
		runner.Queued:             "pending",
		runner.Running:            "pending",
		runner.Passed:             "success",
		runner.PassedWithFailures: "success",
		runner.Failed:             "failure",
		runner.Killed:             "failure",
		runner.TimedOut:           "failure",
	}

	PullRequestActions = map[string]struct{}{
		"opened":      {},
		"reopened":    {},
		"unlocked":    {},
		"synchronize": {},
	}
)

func decodeError(r io.Reader) Error {
	err := Error{}
	json.NewDecoder(r).Decode(&err)
	return err
}

// New returns a new Client to the GitHub REST API. If the given endpoint is
// empty then the default "https://api.github.com" will be used. This will
// set the following scopes to request: "admin:repo_hook", "read:org", "repo".
func New(host, endpoint, secret, clientId, clientSecret string) *Client {
	if endpoint == "" {
		endpoint = "https://api.github.com"
	}

	url := strings.Replace(endpoint, "api.", "", 1)

	b := make([]byte, 16)
	rand.Read(b)

	cfg := oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  host + "/oauth/github",
		Scopes:       []string{"admin:repo_hook", "read:org", "repo"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  url + "/login/oauth/authorize",
			TokenURL: url + "/login/oauth/access_token",
		},
	}

	return &Client{
		BaseClient: provider.BaseClient{
			Config:      cfg,
			Host:        host,
			APIEndpoint: endpoint,
			State:       hex.EncodeToString(b),
			Secret:      secret,
		},
	}
}

func (g *Client) findHook(tok, name, url string) (int64, error) {
	resp, err := g.Get(tok, "/repos/"+name+"/hooks")

	if err != nil {
		return 0, errors.Err(err)
	}

	defer resp.Body.Close()

	hooks := make([]struct {
		ID     int64
		Config struct {
			URL string
		}
	}, 0)

	json.NewDecoder(resp.Body).Decode(&hooks)

	var id int64

	for _, hook := range hooks {
		if hook.Config.URL == url {
			id = hook.ID
			break
		}
	}
	return id, nil
}

// VerifyRequest implements the provider.Client interface.
func (g *Client) VerifyRequest(r io.Reader, signature string) ([]byte, error) {
	if len(signature) != signatureLength {
		return nil, provider.ErrInvalidSignature
	}

	b, err := ioutil.ReadAll(r)

	if err != nil {
		return nil, errors.Err(err)
	}

	actual := make([]byte, 20)

	hex.Decode(actual, []byte(signature[5:]))

	expected := hmac.New(sha1.New, []byte(g.Secret))
	expected.Write(b)

	if !hmac.Equal(expected.Sum(nil), actual) {
		return nil, provider.ErrInvalidSignature
	}
	return b, nil
}

// Repos implements the provider.Client interface.
func (g *Client) Repos(tok string, page int64) ([]*provider.Repo, database.Paginator, error) {
	resp, err := g.Get(tok, "/user/repos?sort=updated&page="+strconv.FormatInt(page, 10))

	if err != nil {
		return nil, database.Paginator{}, errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, database.Paginator{}, errors.New("unexpected http status: " + resp.Status)
	}

	repos := make([]Repo, 0)

	json.NewDecoder(resp.Body).Decode(&repos)

	rr := make([]*provider.Repo, 0, len(repos))

	for _, r := range repos {
		rr = append(rr, &provider.Repo{
			ProviderUserID: r.Owner.ID,
			RepoID:         r.ID,
			Name:           r.FullName,
			Href:           r.HTMLURL,
		})
	}

	next, prev := provider.GetNextAndPrev(resp.Header.Get("Link"))

	p := database.Paginator{
		Next:  next,
		Prev:  prev,
		Page:  page,
		Pages: []int64{next, prev},
	}
	return rr, p, nil
}

// Groups implements the provider.Client interface. For GitHub this will return
// a slice of the organization IDs the user is either an owner of, or a member
// of.
func (g *Client) Groups(tok string) ([]int64, error) {
	resp, err := g.Get(tok, "/user/orgs")

	if err != nil {
		return nil, errors.Err(err)
	}

	defer resp.Body.Close()

	orgs := make([]struct {
		ID int64
	}, 0)

	json.NewDecoder(resp.Body).Decode(&orgs)

	ids := make([]int64, 0, len(orgs))

	for _, org := range orgs {
		ids = append(ids, org.ID)
	}
	return ids, nil
}

// ToggleRepo implementas the provider.Client interface.
func (g *Client) ToggleRepo(tok string, r *provider.Repo) error {
	if !r.Enabled {
		body := map[string]interface{}{
			"config": map[string]interface{}{
				"url":          g.Host + "/hook/github",
				"secret":       g.Secret,
				"content_type": "json",
				"insecure_ssl": 0,
			},
			"events": []string{"push", "pull_request"},
		}

		buf := &bytes.Buffer{}

		json.NewEncoder(buf).Encode(body)

		resp, err := g.Post(tok, "/repos/"+r.Name+"/hooks", buf)

		if err != nil {
			return errors.Err(err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			err := decodeError(resp.Body)

			if resp.StatusCode == http.StatusUnprocessableEntity {
				return provider.ErrLocalhost
			}

			if err.Has("Hook already exists on this repository") {
				hookId, err := g.findHook(tok, r.Name, g.Host+"/hook/github")

				if err != nil {
					return errors.Err(err)
				}

				r.HookID = sql.NullInt64{
					Int64: hookId,
					Valid: hookId > 0,
				}
				r.Enabled = r.HookID.Valid
				return nil
			}
			return errors.Err(err)
		}

		hook := struct {
			ID int64
		}{}

		json.NewDecoder(resp.Body).Decode(&hook)

		r.HookID = sql.NullInt64{
			Int64: hook.ID,
			Valid: true,
		}
		r.Enabled = true
		return nil
	}

	resp, err := g.Delete(tok, "/repos/"+r.Name+"/hooks/"+strconv.FormatInt(r.HookID.Int64, 10))

	if err != nil {
		return errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return decodeError(resp.Body)
	}

	r.HookID = sql.NullInt64{
		Int64: 0,
		Valid: false,
	}
	r.Enabled = false
	return nil
}

// SetCommitStatus implements the provider.Client interface.
func (g *Client) SetCommitStatus(tok string, r *provider.Repo, status runner.Status, url, sha string) error {
	body := map[string]string{
		"state":       states[status],
		"target_url":  url,
		"description": provider.StatusDescriptions[status],
		"context":     "continuous-integration/djinn-ci",
	}

	buf := &bytes.Buffer{}

	json.NewEncoder(buf).Encode(body)

	resp, err := g.Post(tok, "/repos/"+r.Name+"/statuses/"+sha, buf)

	if err != nil {
		return errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return errors.Err(decodeError(resp.Body))
	}
	return nil
}

// Error returns a formatted string of an error from the GitHub API.
func (e Error) Error() string {
	if len(e.Errors) > 0 {
		s := e.Message + ": "

		for i, err := range e.Errors {
			s += err["message"]

			if i != len(e.Errors)-1 {
				s += ", "
			}
		}
		return s
	}
	return e.Message
}

// Has reports whether or not the given error string exists in the underlying
// error.
func (e Error) Has(err1 string) bool {
	for _, err := range e.Errors {
		if err["message"] == err1 {
			return true
		}
	}
	return false
}
