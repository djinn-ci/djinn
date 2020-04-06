package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/oauth2"
)

type GitHub struct {
	client
}

var (
	_ oauth2.Provider = (*GitHub)(nil)

	githubURL    = "https://api.github.com"
	githubScopes = []string{
		"repo",
		"admin:repo_hook",
	}
)

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

	dec := json.NewDecoder(respGet.Body)
	dec.Decode(&repo)

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

		enc := json.NewEncoder(buf)
		enc.Encode(body)

		url := fmt.Sprintf("%s/repos/%s/%s/hooks", g.Endpoint, repo.Owner.Login, repo.Name)
		respPost, err := g.Post(string(tok), url, buf)

		if err != nil {
			return 0, errors.Err(err)
		}

		defer respPost.Body.Close()

		if respPost.StatusCode != http.StatusCreated {
			return 0, errors.New("unexpected http status "+respPost.Status)
		}

		hook := struct{
			ID int64
		}{}

		dec := json.NewDecoder(respPost.Body)
		dec.Decode(&hook)
		return hook.ID, nil
	}

	url := fmt.Sprintf("%s/repos/%s/%s/hooks/%v", g.Endpoint, repo.Owner.Login, repo.Name, hookId)
	respDelete, err := g.Delete(string(tok), url)

	if err != nil {
		return 0, errors.Err(err)
	}

	defer respDelete.Body.Close()

	if respDelete.StatusCode != http.StatusNoContent {
		return 0, errors.New("unexpected http status "+respDelete.Status)
	}
	return hookId, nil
}

func (g GitHub) Repos(tok []byte) ([]oauth2.Repo, error) {
	resp, err := g.Get(string(tok), g.Endpoint+"/user/repositories?sort=updated")

	if err != nil {
		return []oauth2.Repo{}, errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []oauth2.Repo{}, errors.New("unexpected status code "+resp.Status)
	}

	repos := make([]struct{
		ID       int64
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	}, 0)

	dec := json.NewDecoder(resp.Body)
	dec.Decode(&repos)

	rr := make([]oauth2.Repo, 0, len(repos))

	for _, repo := range repos {
		rr = append(rr, oauth2.Repo{
			ID:   repo.ID,
			Name: repo.FullName,
			Href: repo.HTMLURL,
		})
	}
	return rr, nil
}

func (g GitHub) Revoke(tok []byte) error {
	auth := g.Config.ClientID+":"+g.Config.ClientSecret

	url := fmt.Sprintf("%s/applications/%s/token/%s", g.Endpoint, g.Config.ClientID, string(tok))
	req, err := http.NewRequest("DELETE", url, nil)

	if err != nil {
		return errors.Err(err)
	}

	req.Header.Set("Authorization", "basic "+base64.StdEncoding.EncodeToString([]byte(auth)))

	cli := &http.Client{}

	resp, err := cli.Do(req)

	if err != nil {
		return errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return errors.New("unexpecred status code "+resp.Status)
	}
	return nil
}
