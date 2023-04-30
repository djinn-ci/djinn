package github

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"djinn-ci.com/auth"
	"djinn-ci.com/auth/oauth2"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/manifest"
	"djinn-ci.com/provider"
	"djinn-ci.com/runner"
)

type user struct {
	ID    int64
	Login string
}

type repo struct {
	ID          int64
	Owner       user
	FullName    string `json:"full_name"`
	HTMLURL     string `json:"html_url"`
	ContentsURL string `json:"contents_url"`
}

type pushEvent struct {
	Ref        string
	Repo       repo `json:"repository"`
	HeadCommit struct {
		ID      string
		URL     string
		Message string
		Author  map[string]string
	} `json:"head_commit"`
}

type pullRequestEvent struct {
	Action      string
	Number      int64
	PullRequest struct {
		Title   string
		User    user
		HTMLURL string `json:"html_url"`
		Head    struct {
			Sha  string
			User user
			Repo repo
		}
		Base struct {
			Ref  string
			User user
			Repo repo
		}
	} `json:"pull_request"`
}

type ghError struct {
	Request *http.Request `json:"-"`
	Message string
	Errors  []map[string]string
}

func decodeError(req *http.Request, r io.Reader) *ghError {
	err := ghError{
		Request: req,
	}

	json.NewDecoder(r).Decode(&err)
	return &err
}

func (e *ghError) Error() string {
	var buf bytes.Buffer

	buf.WriteString(e.Request.Method + " ")
	buf.WriteString(e.Request.URL.String() + ": ")
	buf.WriteString(e.Message)

	if len(e.Errors) > 0 {
		for i, err := range e.Errors {
			buf.WriteString(err["message"])

			if i != len(e.Errors)-1 {
				buf.WriteString(", ")
			}
		}
	}
	return buf.String()
}

func (e *ghError) has(msg string) bool {
	for _, err := range e.Errors {
		if err["message"] == msg {
			return true
		}
	}
	return false
}

type Client struct {
	*provider.BaseClient
}

const (
	defaultEndpoint = "https://api.github.com"
	oauth2Endpoint  = "https://github.com"
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

	cfg.Endpoint.AuthURL = endpoint.OAuth + "/login/oauth/authorize"
	cfg.Endpoint.TokenURL = endpoint.OAuth + "/login/oauth/access_token"

	cfg.RedirectURL = host + "/oauth/github"
	cfg.Scopes = []string{"admin:repo_hook", "read:org", "repo"}

	cfg.State = hex.EncodeToString(b)
	cfg.GetUser = func(cli *http.Client, tok *oauth2.Token) (*auth.User, error) {
		u, err := oauth2.GetUser("github", endpoint.API+"/user")(cli, tok)

		if err != nil {
			return nil, errors.Err(err)
		}

		u.Username = u.RawData["login"].(string)
		return u, nil
	}

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
	req, err := c.Get("/user/repos?sort=updated&page=" + strconv.Itoa(page))

	if err != nil {
		return nil, errors.Err(err)
	}

	resp, err := c.Do(req)

	if err != nil {
		return nil, errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github: GET %s unexpected status: %s", req.URL.String(), resp.Status)
	}

	repos := make([]repo, 0)

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

	var prev, next int

	links := provider.ParseLink(resp.Header.Get("Link"))

	if urls, ok := links["next"]; ok {
		url, _ := url.Parse(urls[0])
		next, _ = strconv.Atoi(url.Query().Get("page"))
	}
	if urls, ok := links["prev"]; ok {
		url, _ := url.Parse(urls[0])
		prev, _ = strconv.Atoi(url.Query().Get("page"))
	}

	return &database.Paginator[*provider.Repo]{
		Items: rr,
		Pages: []int{prev, page, next},
	}, nil
}

func (c Client) Groups() ([]int64, error) {
	req, err := c.Get("/user/orgs")

	if err != nil {
		return nil, errors.Err(err)
	}

	resp, err := c.Do(req)

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

func (c Client) findHook(name, url string) (int64, error) {
	req, err := c.Get("/repos/" + name + "/hooks")

	if err != nil {
		return 0, errors.Err(err)
	}

	resp, err := c.Do(req)

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

	for _, hook := range hooks {
		if hook.Config.URL == url {
			return hook.ID, nil
		}
	}
	return 0, errors.New("github: webhook not found: " + url)
}

func (c Client) ToggleWebhook(r *provider.Repo) error {
	if !r.Enabled {
		var buf bytes.Buffer

		json.NewEncoder(&buf).Encode(map[string]any{
			"config": map[string]any{
				"url":          c.Host + "/hook/github",
				"secret":       c.Secret,
				"content_type": "json",
				"insecure_ssl": 0,
			},
			"events": []string{"push", "pull_request"},
		})

		req, err := c.Post("/repos/"+r.Name+"/hooks", &buf)

		if err != nil {
			return errors.Err(err)
		}

		resp, err := c.Do(req)

		if err != nil {
			return errors.Err(err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			err := decodeError(req, resp.Body)

			if resp.StatusCode == http.StatusUnprocessableEntity {
				return provider.ErrLocalhost
			}

			if err.has("Hook already exists on this repository") {
				hookId, err := c.findHook(r.Name, c.Host+"/hook/github")

				if err != nil {
					return errors.Err(err)
				}

				r.HookID = database.Null[int64]{
					Elem:  hookId,
					Valid: hookId > 0,
				}
				r.Enabled = r.HookID.Valid
				return nil
			}
			return errors.Err(err)
		}

		var hook struct {
			ID int64
		}

		json.NewDecoder(resp.Body).Decode(&hook)

		r.HookID = database.Null[int64]{
			Elem:  hook.ID,
			Valid: true,
		}
		r.Enabled = true
		return nil
	}

	req, err := c.Delete("/repos/" + r.Name + "/hooks/" + strconv.Itoa(int(r.HookID.Elem)))

	if err != nil {
		return errors.Err(err)
	}

	resp, err := c.Do(req)

	if err != nil {
		return errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return decodeError(req, resp.Body)
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
		runner.Running:            "pending",
		runner.Passed:             "success",
		runner.PassedWithFailures: "success",
		runner.Failed:             "failure",
		runner.Killed:             "failure",
		runner.TimedOut:           "failure",
	}

	var buf bytes.Buffer

	json.NewEncoder(&buf).Encode(map[string]string{
		"state":       states[status],
		"target_url":  url,
		"description": provider.StatusDescriptions[status],
		"context":     "continuous-integration/djinn-ci",
	})

	req, err := c.Post("/repos/"+r.Name+"/statuses/"+sha, &buf)

	if err != nil {
		return errors.Err(err)
	}

	resp, err := c.Do(req)

	if err != nil {
		return errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return errors.Err(decodeError(req, resp.Body))
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
		if it["type"] == "dir" {
			continue
		}
		if strings.HasSuffix(it["name"], ".yml") {
			urls = append(urls, it["url"])
		}
	}

	mm, err := c.LoadManifests(urls, provider.UnmarshalBase64Manifest)

	if err != nil {
		return nil, errors.Err(err)
	}
	return mm, nil
}

const signatureLength = 45

func (c Client) Verify(r *http.Request) ([]byte, error) {
	signature := r.Header.Get("X-Hub-Signature")

	if len(signature) != signatureLength {
		return nil, provider.ErrInvalidSignature
	}

	b, err := io.ReadAll(r.Body)

	if err != nil {
		return nil, errors.Err(err)
	}

	actual := make([]byte, 20)

	hex.Decode(actual, []byte(signature[5:]))

	expected := hmac.New(sha1.New, []byte(c.Secret))
	expected.Write(b)

	if !hmac.Equal(expected.Sum(nil), actual) {
		return nil, provider.ErrInvalidSignature
	}
	return b, nil
}
