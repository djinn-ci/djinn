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

	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/provider"
	"github.com/andrewpillar/thrall/runner"

	"golang.org/x/oauth2"
)

type GitLab struct {
	oauth2.Config
	provider.Client
}

type Repo struct {
	ID     int64
	Name   string `json:"path_with_namespace"`
	URL    string
	WebURL string `json:"web_url"`
}

type PushEvent struct {
	After   string
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
	ID   int64
	Name string
}

type MergeRequestEvent struct {
	User    User
	Project Repo
	Attrs   struct {
		ID              int64  `json:"id"`
		TargetBranch    string `json:"target_branch"`
		SourceProjectID int64  `json:"source_project_id"`
		AuthorID        int64  `json:"author_id"`
		Title           string
		Source          struct {
			WebURL string `json:"web_url"`
		}
		LastCommit      struct {
			ID string
		} `json:"last_commit"`
		URL    string
		Action string
	} `json:"object_attributes"`
}

var (
	_ provider.Interface = (*GitLab)(nil)
)

func New(host, endpoint, secret, clientId, clientSecret string) *GitLab {
	if endpoint == "" {
		endpoint = "https://gitlab.com/api/v4"
	}

	url := strings.Replace(endpoint, "/api/v4", "", 1)

	b := make([]byte, 16)
	rand.Read(b)

	cfg := oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  host + "/oauth/gitlab",
		Scopes:       []string{"api"},
		Endpoint:     oauth2.Endpoint{
			AuthURL:  url + "/oauth/authorize",
			TokenURL: url + "/oauth/token",
		},
	}

	cli := provider.Client{
		Host:        host,
		APIEndpoint: endpoint,
		State:       hex.EncodeToString(b),
		Secret:      secret,
	}

	return &GitLab{
		Config: cfg,
		Client: cli,
	}
}

func (g *GitLab) VerifyRequest(r io.Reader, signature string) ([]byte, error) {
	b, _ := ioutil.ReadAll(r)

	if g.Secret != signature {
		return nil, provider.ErrInvalidSignature
	}
	return b, nil
}

func (g *GitLab) Repos(tok string, page int64) ([]*provider.Repo, database.Paginator, error) {
	resp0, err := g.Get(tok, "/user")

	if err != nil {
		return nil, database.Paginator{}, errors.Err(err)
	}

	defer resp0.Body.Close()

	if resp0.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(resp0.Body)
		return nil, database.Paginator{}, errors.New(string(b))
	}

	u := User{}

	json.NewDecoder(resp0.Body).Decode(&u)

	suid := strconv.FormatInt(u.ID, 10)
	spage := strconv.FormatInt(page, 10)

	resp, err := g.Get(tok, "/users/" + suid + "/projects?simple=true&order_by=updated&page=" + spage)

	if err != nil {
		return nil, database.Paginator{}, errors.Err(err)
	}

	defer resp.Body.Close()

	repos := make([]Repo, 0)

	json.NewDecoder(resp.Body).Decode(&repos)

	rr := make([]*provider.Repo, 0, len(repos))

	for _, r := range repos {
		rr = append(rr, &provider.Repo{
			RepoID: r.ID,
			Name:   r.Name,
			Href:   r.WebURL,
		})
	}

	next, prev := provider.GetNextAndPrev(resp.Header.Get("Link"))

	p := database.Paginator{
		Next:  next,
		Prev:  prev,
		Pages: []int64{next, prev},
	}
	return rr, p, nil
}

func (g *GitLab) ToggleRepo(tok string, r *provider.Repo) error {
	id := strconv.FormatInt(r.RepoID, 10)

	if !r.Enabled {
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

		resp, err := g.Post(tok, "/projects/" + id, buf)

		if err != nil {
			return errors.Err(err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			b, _ := ioutil.ReadAll(resp.Body)
			return errors.New(string(b))
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

	resp, err := g.Delete(tok, "/projects/" + id + "/hooks/" + strconv.FormatInt(r.HookID.Int64, 10))

	if err != nil {
		return errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		b, _ := ioutil.ReadAll(resp.Body)
		return errors.New(string(b))
	}

	r.HookID = sql.NullInt64{
		Int64: 0,
		Valid: false,
	}
	r.Enabled = false
	return nil
}

func (g *GitLab) SetCommitStatus(tok string, r *provider.Repo, status runner.Status, url, sha string) error {
	return nil
}