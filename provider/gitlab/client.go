// Package gitlab provides an implementation of the provider.Client interface
// for the GitLab REST API.
package gitlab

import (
	"bytes"
	"crypto/rand"
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

type Repo struct {
	ID     int64
	Name   string `json:"path_with_namespace"`
	URL    string
	WebURL string `json:"web_url"`
	Owner  struct {
		ID int64
	}
	Namespace struct {
		ID   int64
		Kind string
	}
}

type PushEvent struct {
	Ref     string
	UserID  int64 `json:"user_id"`
	Project Repo
	Commits []struct {
		ID      string
		Message string
		URL     string
		Author  map[string]string
	}
}

type User struct {
	ID       int64
	Username string
}

type MergeRequestEvent struct {
	User    User
	Project Repo
	Attrs   struct {
		ID              int64  `json:"iid"`
		TargetBranch    string `json:"target_branch"`
		SourceProjectID int64  `json:"source_project_id"`
		TargetProjectID int64  `json:"target_project_id"`
		AuthorID        int64  `json:"author_id"`
		Title           string
		Source          struct {
			WebURL string `json:"web_url"`
		}
		LastCommit struct {
			ID string
		} `json:"last_commit"`
		URL    string
		Action string
	} `json:"object_attributes"`
}

// Error reports an error that occurred when interacting with the GitLab API.
// This captures the status code received from the API, and the body of the
// response.
type Error struct {
	StatusCode int
	Body       io.Reader
}

func (e *Error) Error() string {
	b, _ := io.ReadAll(e.Body)
	return string(b)
}

var (
	_ provider.Client = (*Client)(nil)

	states = map[runner.Status]string{
		runner.Queued:             "pending",
		runner.Running:            "running",
		runner.Passed:             "success",
		runner.PassedWithFailures: "success",
		runner.Failed:             "failed",
		runner.Killed:             "canceled",
		runner.TimedOut:           "canceled",
	}
)

const defaultEndpoint = "https://gitlab.com/api/v4"

// New returns a new Client to the GitLab REST API. If the given endpoint is
// empty then the default "https://gitlab.com/api/v4" will be used. This will
// set the following scopes to request: "api".
func New(p provider.ClientParams) *Client {
	if p.Endpoint == "" {
		p.Endpoint = defaultEndpoint
	}

	url := strings.Replace(p.Endpoint, "/api/v4", "", 1)

	b := make([]byte, 16)
	rand.Read(b)

	cfg := oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		RedirectURL:  p.Host + "/oauth/gitlab",
		Scopes:       []string{"api"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  url + "/oauth/authorize",
			TokenURL: url + "/oauth/token",
		},
	}

	return &Client{
		BaseClient: provider.BaseClient{
			Config:      cfg,
			Host:        p.Host,
			APIEndpoint: p.Endpoint,
			State:       hex.EncodeToString(b),
			Secret:      p.Secret,
		},
	}
}

// VerifyRequest implements the provider.Client interface.
func (g *Client) VerifyRequest(r io.Reader, signature string) ([]byte, error) {
	b, _ := ioutil.ReadAll(r)

	if g.Secret != signature {
		return nil, provider.ErrInvalidSignature
	}
	return b, nil
}

// Repos implements the provider.Client interface.
func (g *Client) Repos(tok string, page int64) ([]*provider.Repo, database.Paginator, error) {
	spage := strconv.FormatInt(page, 10)

	resp, err := g.Get(tok, "/projects?&membership=true&order_by=updated_at&page="+spage)

	if err != nil {
		return nil, database.Paginator{}, errors.Err(err)
	}

	defer resp.Body.Close()

	repos := make([]Repo, 0)

	json.NewDecoder(resp.Body).Decode(&repos)

	rr := make([]*provider.Repo, 0, len(repos))

	for _, r := range repos {
		providerUserId := r.Namespace.ID

		if r.Namespace.Kind == "user" {
			providerUserId = r.Owner.ID
		}

		rr = append(rr, &provider.Repo{
			ProviderUserID: providerUserId,
			RepoID:         r.ID,
			Name:           r.Name,
			Href:           r.WebURL,
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

// Groups implements the provider.Client interface.
func (g *Client) Groups(tok string) ([]int64, error) {
	resp, err := g.Get(tok, "/groups")

	if err != nil {
		return nil, errors.Err(err)
	}

	defer resp.Body.Close()

	groups := make([]struct {
		ID int64
	}, 0)

	json.NewDecoder(resp.Body).Decode(&groups)

	ids := make([]int64, 0, len(groups))

	for _, group := range groups {
		ids = append(ids, group.ID)
	}
	return ids, nil
}

// ToggleRepo implements the provider.Client interface.
func (g *Client) ToggleRepo(tok string, r *provider.Repo) error {
	id := strconv.FormatInt(r.RepoID, 10)

	if !r.Enabled {
		resp0, err := g.Get(tok, "/projects/"+id+"/hooks")

		if err != nil {
			return errors.Err(err)
		}

		defer resp0.Body.Close()

		hooks := make([]struct {
			ID  int64
			URL string
		}, 0)

		json.NewDecoder(resp0.Body).Decode(&hooks)

		for _, hook := range hooks {
			if hook.URL == g.Host+"/hook/gitlab" {
				r.HookID = sql.NullInt64{
					Int64: hook.ID,
					Valid: hook.ID > 0,
				}
				r.Enabled = r.HookID.Valid
				return nil
			}
		}

		body := map[string]interface{}{
			"id":                       r.RepoID,
			"url":                      g.Host + "/hook/gitlab",
			"push_events":              true,
			"merge_requests_events":    true,
			"enable_ssl_vertification": true,
			"token":                    g.Secret,
		}

		buf := &bytes.Buffer{}

		json.NewEncoder(buf).Encode(body)

		resp, err := g.Post(tok, "/projects/"+id+"/hooks", buf)

		if err != nil {
			return errors.Err(err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			var buf bytes.Buffer
			buf.ReadFrom(resp.Body)

			return &Error{
				StatusCode: resp.StatusCode,
				Body:       &buf,
			}
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

	resp, err := g.Delete(tok, "/projects/"+id+"/hooks/"+strconv.FormatInt(r.HookID.Int64, 10))

	if err != nil {
		return errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)

		return &Error{
			StatusCode: resp.StatusCode,
			Body:       &buf,
		}
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
	body := map[string]interface{}{
		"id":          r.RepoID,
		"sha":         sha,
		"state":       states[status],
		"target_url":  url,
		"description": provider.StatusDescriptions[status],
		"context":     "continuous-integration/djinn-ci",
	}

	var buf bytes.Buffer

	json.NewEncoder(&buf).Encode(body)

	resp, err := g.Post(tok, "/projects/"+strconv.FormatInt(r.RepoID, 10)+"/statuses/"+sha, &buf)

	if err != nil {
		return errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)

		return &Error{
			StatusCode: resp.StatusCode,
			Body:       &buf,
		}
	}
	return nil
}
