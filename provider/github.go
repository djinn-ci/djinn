package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/oauth2"
)

// GitHub provides an oauth2.Provider implementation for the GitHub REST API.
type GitHub struct {
	client
}

type githubError struct {
	Message string
	Errors  []map[string]string
}

var (
	_ oauth2.Provider = (*GitHub)(nil)

	githubURL    = "https://api.github.com"
	githubScopes = []string{
		"repo",
		"admin:repo_hook",
	}
)

func (e githubError) String() string {
	buf := bytes.Buffer{}
	buf.WriteString(e.Message + ": ")
	buf.WriteString(e.Errors[0]["message"])
	return buf.String()
}

// ToggleRepo will either add or remove the webhook to the repository of the
// given ID depending on whether it was previously enabled as determined by
// the given callback. The new webhook will trigger on push and pull_request
// events.
func (g GitHub) ToggleRepo(tok []byte, id int64, enabled func(int64) (int64, bool, error)) (int64, error) {
	respGet, err := g.Get(string(tok), fmt.Sprintf("%s/repositories/%v", g.Endpoint, id))

	if err != nil {
		return 0, errors.Err(err)
	}

	defer respGet.Body.Close()

	if respGet.StatusCode != http.StatusOK {
		return 0, errors.New("failed to get repository "+respGet.Status)
	}

	repo := struct{
		Name  string
		Owner struct{
			Login string
		}
	}{}

	json.NewDecoder(respGet.Body).Decode(&repo)

	hookId, ok, err := enabled(id)

	if err != nil {
		return 0, errors.Err(err)
	}

	if !ok {
		body := map[string]interface{}{
			"config": map[string]interface{}{
				"url":          g.hookEndpoint,
				"secret":       g.secret,
				"content_type": "json",
				"insecure_ssl": 0,
			},
			"events": []string{"push", "pull_request"},
		}

		buf := &bytes.Buffer{}

		json.NewEncoder(buf).Encode(body)

		url := fmt.Sprintf("%s/repos/%s/%s/hooks", g.Endpoint, repo.Owner.Login, repo.Name)
		respPost, err := g.Post(string(tok), url, buf)

		if err != nil {
			return 0, errors.Err(err)
		}

		defer respPost.Body.Close()

		if respPost.StatusCode != http.StatusCreated {
			ghErr := githubError{}

			json.NewDecoder(respPost.Body).Decode(&ghErr)
			return 0, errors.New(ghErr.String())
		}

		hook := struct{
			ID int64
		}{}

		json.NewDecoder(respPost.Body).Decode(&hook)
		return hook.ID, nil
	}

	url := fmt.Sprintf("%s/repos/%s/%s/hooks/%v", g.Endpoint, repo.Owner.Login, repo.Name, hookId)
	respDelete, err := g.Delete(string(tok), url)

	if err != nil {
		return 0, errors.Err(err)
	}

	defer respDelete.Body.Close()

	if respDelete.StatusCode != http.StatusNoContent {
		ghErr := githubError{}

		json.NewDecoder(respDelete.Body).Decode(&ghErr)
		return 0, errors.New(ghErr.String())
	}
	return 0, nil
}

// Repos returns the repos from GitHub ordered by when they were last updated.
func (g GitHub) Repos(tok []byte, page int64) (oauth2.Repos, error) {
	resp, err := g.Get(string(tok), fmt.Sprintf("%s/user/repos?sort=updated&page=%v", g.Endpoint, page))

	if err != nil {
		return oauth2.Repos{}, errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return oauth2.Repos{}, errors.New("unexpected status code "+resp.Status)
	}

	next, prev := getNextAndPrev(resp.Header.Get("Link"))

	repos := oauth2.Repos{
		Next:  next,
		Prev:  prev,
	}

	items := make([]struct{
		ID       int64
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	}, 0)

	json.NewDecoder(resp.Body).Decode(&items)

	for _, item := range items {
		repos.Items = append(repos.Items, oauth2.Repo{
			ID:   item.ID,
			Name: item.FullName,
			Href: item.HTMLURL,
		})
	}
	return repos, nil
}
