package gitlab

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"djinn-ci.com/auth/oauth2"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/manifest"
	"djinn-ci.com/provider"
	"djinn-ci.com/runner"
)

type glError struct {
	req        *http.Request
	statusCode int
	body       io.Reader
}

func (e *glError) Error() string {
	var buf bytes.Buffer

	buf.WriteString(e.req.Method + " ")
	buf.WriteString(e.req.URL.String() + ": ")

	buf.ReadFrom(e.body)

	return buf.String()
}

type repo struct {
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

type pushEvent struct {
	Ref     string
	UserID  int64 `json:"user_id"`
	Project repo
	Commits []struct {
		ID      string
		Message string
		URL     string
		Author  map[string]string
	}
}

type user struct {
	ID       int64
	Username string
}

type mergeRequestEvent struct {
	User    user
	Project repo
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

type Client struct {
	*provider.BaseClient
}

const (
	defaultEndpoint = "https://gitlab.com/api/v4"
	oauth2Endpoint  = "https://gitlab.com"
)

func Oauth2Config(host, clientId, clientSecret string, endpoint oauth2.Endpoint) *oauth2.Config {
	if endpoint.OAuth == "" {
		endpoint.OAuth = oauth2Endpoint
	}

	if endpoint.API == "" {
		endpoint.API = defaultEndpoint
	}

	var cfg oauth2.Config

	b := make([]byte, 16)
	rand.Read(b)

	cfg.ClientID = clientId
	cfg.ClientSecret = clientSecret

	cfg.Endpoint.AuthURL = endpoint.OAuth + "/oauth/authorize"
	cfg.Endpoint.TokenURL = endpoint.OAuth + "/oauth/token"

	cfg.RedirectURL = host + "/oauth/gitlab?auth_mech=oauth2.gitlab"
	cfg.Scopes = []string{"api"}

	cfg.State = hex.EncodeToString(b)
	cfg.GetUser = oauth2.GetUser("gitlab", endpoint.API+"/user")

	return &cfg
}

func New(cli *provider.BaseClient) provider.Client {
	if cli.Endpoint == "" {
		cli.Endpoint = defaultEndpoint
	}

	return Client{
		BaseClient: cli,
	}
}

func (c Client) Repos(page int) (*database.Paginator[*provider.Repo], error) {
	req, err := c.Get("/projects?membership=true&order_by=updated_at&page=" + strconv.Itoa(page))

	if err != nil {
		return nil, errors.Err(err)
	}

	resp, err := c.Do(req)

	if err != nil {
		return nil, errors.Err(err)
	}

	defer resp.Body.Close()

	repos := make([]repo, 0)

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

	var prev, next int

	links := provider.ParseLink(resp.Header.Get("Link"))

	if urls, ok := links["next"]; ok {
		url, _ := url.Parse(urls[0])
		next, _ = strconv.Atoi(url.Query().Get("page"))
	}
	if urls, ok := links["prev"]; ok {
		url, _ := url.Parse(urls[0])
		next, _ = strconv.Atoi(url.Query().Get("page"))
	}

	return &database.Paginator[*provider.Repo]{
		Items: rr,
		Pages: []int{prev, int(page), next},
	}, nil
}

func (c Client) Groups() ([]int64, error) {
	req, err := c.Get("/groups")

	if err != nil {
		return nil, errors.Err(err)
	}

	resp, err := c.Do(req)

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

func (c Client) ToggleWebhook(r *provider.Repo) error {
	id := strconv.FormatInt(r.RepoID, 10)

	if !r.Enabled {
		req, err := c.Get("/projects/" + id + "/hooks")

		if err != nil {
			return errors.Err(err)
		}

		resp, err := c.Do(req)

		if err != nil {
			return errors.Err(err)
		}

		defer resp.Body.Close()

		hooks := make([]struct {
			ID  int64
			URL string
		}, 0)

		json.NewDecoder(resp.Body).Decode(&hooks)

		for _, hook := range hooks {
			if hook.URL == c.Host+"/hook/gitlab" {
				r.HookID = database.Null[int64]{
					Elem:  hook.ID,
					Valid: hook.ID > 0,
				}
				r.Enabled = r.HookID.Valid
				return nil
			}
		}

		var buf bytes.Buffer

		json.NewEncoder(&buf).Encode(map[string]any{
			"id":                       r.RepoID,
			"url":                      c.Host + "/hook/gitlab",
			"push_events":              true,
			"merge_requests_events":    true,
			"enable_ssl_vertification": true,
			"token":                    c.Secret,
		})

		req, err = c.Post("/projects/"+id+"/hooks", &buf)

		if err != nil {
			return errors.Err(err)
		}

		resp2, err := c.Do(req)

		if err != nil {
			return errors.Err(err)
		}

		defer resp2.Body.Close()

		var hook struct {
			ID int64
		}

		json.NewDecoder(resp2.Body).Decode(&hook)

		r.HookID = database.Null[int64]{
			Elem:  hook.ID,
			Valid: true,
		}
		r.Enabled = true
		return nil
	}

	req, err := c.Delete("/projects/" + id + "/hooks/" + strconv.FormatInt(r.HookID.Elem, 10))

	if err != nil {
		return errors.Err(err)
	}

	resp, err := c.Do(req)

	if err != nil {
		return errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)

		return &glError{
			req:        req,
			statusCode: resp.StatusCode,
			body:       &buf,
		}
	}

	r.HookID = database.Null[int64]{
		Elem:  0,
		Valid: false,
	}
	r.Enabled = false
	return nil
}

func (c Client) SetCommitStatus(r *provider.Repo, status runner.Status, url, sha string) error {
	states := map[runner.Status]string{
		runner.Queued:             "pending",
		runner.Running:            "running",
		runner.Passed:             "success",
		runner.PassedWithFailures: "success",
		runner.Failed:             "failed",
		runner.Killed:             "canceled",
		runner.TimedOut:           "canceled",
	}

	var buf bytes.Buffer

	json.NewEncoder(&buf).Encode(map[string]any{
		"id":          r.RepoID,
		"sha":         sha,
		"state":       states[status],
		"target_url":  url,
		"description": provider.StatusDescriptions[status],
		"context":     "continuous-integration/djinn-ci",
	})

	req, err := c.Post("/projects/"+strconv.FormatInt(r.RepoID, 10)+"/statuses/"+sha, &buf)

	if err != nil {
		return errors.Err(err)
	}

	resp, err := c.Do(req)

	if err != nil {
		return errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)

		return &glError{
			req:        req,
			statusCode: resp.StatusCode,
			body:       &buf,
		}
	}
	return nil
}

func (c Client) Manifests(root, ref string) ([]manifest.Manifest, error) {
	req, err := c.Get(root + "?ref=" + ref)

	if err != nil {
		return nil, errors.Err(err)
	}

	resp, err := c.Do(req)

	if err != nil {
		return nil, errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	items := make([]map[string]string, 0)
	json.NewDecoder(resp.Body).Decode(&items)

	urls := make([]string, 0, len(items))

	for _, it := range items {
		if it["type"] == "tree" {
			continue
		}
		if path := it["path"]; strings.HasSuffix(path, ".yml") {
			urls = append(urls, root+"/repository/files/"+url.QueryEscape(path)+"?ref="+ref)
		}
	}

	mm, err := c.LoadManifests(urls, provider.UnmarshalBase64Manifest)

	if err != nil {
		return nil, errors.Err(err)
	}
	return mm, nil
}

func (c Client) Verify(r *http.Request) ([]byte, error) {
	b, err := io.ReadAll(r.Body)

	if err != nil {
		return nil, errors.Err(err)
	}

	if c.Secret != r.Header.Get("X-Gitlab-Token") {
		return nil, provider.ErrInvalidSignature
	}
	return b, nil
}
