package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/oauth2"
)

// GitLab provides an oauth2.Provider implementation for the GitLab REST API.
type GitLab struct {
	client
}

var (
	_ oauth2.Provider = (*GitLab)(nil)

	gitlabURL    = "https://gitlab.com"
	gitlabScopes = []string{"api"}
)

// ToggleRepo will either add or remove the webhook to the repository of the
// given ID depending on whether it was previously enabled as determined by
// the given callback. The new webhook will trigger on push_events and
// merge_requests_events.
func (g GitLab) ToggleRepo(tok []byte, id int64, enabled func(int64) (int64, bool, error)) (int64, error) {
	hookId, ok, err := enabled(id)

	if err != nil {
		return 0, errors.Err(err)
	}

	if !ok {
		body := map[string]interface{}{
			"id":                      id,
			"url":                     g.hookEndpoint,
			"push_events":             true,
			"merge_requests_events":   true,
			"enable_ssl_verification": true,
			"token":                   g.secret,
		}

		buf := &bytes.Buffer{}

		enc := json.NewEncoder(buf)
		enc.Encode(body)

		resp, err := g.Post(string(tok), fmt.Sprintf("%s/projects/%v/hooks", g.Endpoint, id), buf)

		if err != nil {
			return 0, errors.Err(err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			return 0, errors.New("unexpected http status " + resp.Status)
		}

		hook := struct {
			ID int64
		}{}

		dec := json.NewDecoder(resp.Body)
		dec.Decode(&hook)
		return hook.ID, nil
	}

	url := fmt.Sprintf("%s/projects/%v/hooks/%v", g.Endpoint, id, hookId)
	resp, err := g.Delete(string(tok), url)

	if err != nil {
		return 0, errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return 0, errors.New("unexpected http status " + resp.Status)
	}
	return 0, nil
}

// Repos returns the projects from GitLab ordered by when they were last
// updated. This will first make a callout to the /user endpoint to get the
// user ID that can then be given to the /users/:id/projects endpoint.
func (g GitLab) Repos(tok []byte, page int64) (oauth2.Repos, error) {
	userResp, err := g.Get(string(tok), g.Endpoint+"/user")

	if err != nil {
		return oauth2.Repos{}, errors.Err(err)
	}

	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		return oauth2.Repos{}, errors.New("unexpected http status " + userResp.Status)
	}

	u := struct {
		ID int64
	}{}

	json.NewDecoder(userResp.Body).Decode(&u)

	url := fmt.Sprintf("%s/users/%v/projects?simple=true&order_by=updated_at&page=%v", g.Endpoint, u.ID, page)
	resp, err := g.Get(string(tok), url)

	if err != nil {
		return oauth2.Repos{}, errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return oauth2.Repos{}, errors.New("unexpected http status " + resp.Status)
	}

	next, prev := getNextAndPrev(resp.Header.Get("Link"))

	repos := oauth2.Repos{
		Next: next,
		Prev: prev,
	}

	items := make([]struct {
		ID     int64
		Name   string `json:"path_with_namespace"`
		WebURL string `json:"web_url"`
	}, 0)

	json.NewDecoder(resp.Body).Decode(&items)

	for _, item := range items {
		repos.Items = append(repos.Items, oauth2.Repo{
			ID:   item.ID,
			Name: item.Name,
			Href: item.WebURL,
		})
	}
	return repos, nil
}
