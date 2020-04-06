package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/oauth2"
)

type GitLab struct {
	client
}

var (
	_ oauth2.Provider = (*GitLab)(nil)

	gitlabURL    = "https://gitlab.com"
	gitlabScopes = []string{"api"}
)

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
			return 0, errors.New("unexpected http status "+resp.Status)
		}

		hook := struct{
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
		return 0, errors.New("unexpected http status "+resp.Status)
	}
	return hookId, nil
}

func (g GitLab) Repos(tok []byte) ([]oauth2.Repo, error) {
	url := fmt.Sprintf("%s/projects?simple=true&order_by=updated_at", g.Endpoint)
	resp, err := g.Get(string(tok), url)

	if err != nil {
		return []oauth2.Repo{}, errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []oauth2.Repo{}, errors.New("unexpected http status "+resp.Status)
	}

	projects := make([]struct{
		ID     int64
		Name   string `json:"path_with_namespace"`
		WebURL string `json:"web_url"`
	}, 0)

	dec := json.NewDecoder(resp.Body)
	dec.Decode(&projects)

	rr := make([]oauth2.Repo, 0, len(projects))

	for _, proj := range projects {
		rr = append(rr, oauth2.Repo{
			ID:   proj.ID,
			Name: proj.Name,
			Href: proj.WebURL,
		})
	}
	return rr, nil
}

func (g GitLab) Revoke(_ []byte) error { return nil }
