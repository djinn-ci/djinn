package oauth2

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"
)

type GitHub struct {
	Client

	hookEndpoint string
	secret       string
}

var (
	githubScopes = []string{
		"repo",
		"admin:repo_hook",
	}

	githubURL = "https://api.github.com"
)

func (g GitHub) Auth(c context.Context, code string, providers model.ProviderStore) error {
	return errors.Err(g.auth(c, "github", code, providers))
}

func (g GitHub) ToggleRepo(p *model.Provider, id int64) error {
	tok, _ := crypto.Decrypt(p.AccessToken)

	respGet, err := g.Get(string(tok), fmt.Sprintf("%s/repositories/%v", g.APIEndpoint, id))

	if err != nil {
		return errors.Err(err)
	}

	defer respGet.Body.Close()

	if respGet.StatusCode != http.StatusOK {
		return errors.Err(errors.New("failed to get repository: " + respGet.Status))
	}

	repo := struct{
		Name  string
		Owner struct {
			Login string
		}
	}{}

	dec := json.NewDecoder(respGet.Body)
	dec.Decode(&repo)

	r, err := p.RepoStore().Get(query.Where("repo_id", "=", id))

	if err != nil {
		return errors.Err(err)
	}

	if !r.Enabled {
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

		respPost, err := g.Post(
			string(tok),
			fmt.Sprintf("%s/repos/%s/%s/hooks", g.APIEndpoint, repo.Owner.Login, repo.Name),
			buf,
		)

		if err != nil {
			return errors.Err(err)
		}

		defer respPost.Body.Close()

		hook := struct{
			ID int64
		}{}

		dec = json.NewDecoder(respPost.Body)
		dec.Decode(&hook)

		return errors.Err(toggleRepo(p, id, hook.ID))
	}

	respDelete, err := g.Delete(
		string(tok),
		fmt.Sprintf("%s/repos/%s/%s/hooks/%v", g.APIEndpoint, repo.Owner.Login, repo.Name, r.HookID),
	)

	if err != nil {
		return errors.Err(err)
	}

	defer respDelete.Body.Close()

	if respDelete.StatusCode != http.StatusNoContent {
		return errors.Err(errors.New("failed to toggle repository hook: " + respDelete.Status))
	}

	return errors.Err(toggleRepo(p, id, r.HookID))
}

func (g GitHub) Repos(p *model.Provider) ([]*model.Repo, error) {
	tok, _ := crypto.Decrypt(p.AccessToken)

	resp, err := g.Get(string(tok), g.APIEndpoint + "/user/repos?sort=updated")

	if err != nil {
		return []*model.Repo{}, errors.Err(err)
	}

	if resp.StatusCode != http.StatusOK {
		return []*model.Repo{}, errors.Err(errors.New("unexpected status code: " + resp.Status))
	}

	defer resp.Body.Close()

	repos := make([]struct{
		ID       int64
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	}, 0)

	dec := json.NewDecoder(resp.Body)
	dec.Decode(&repos)

	rr := make([]*model.Repo, 0, len(repos))

	for _, repo := range repos {
		r := &model.Repo{
			ProviderID: p.ID,
			RepoID:     repo.ID,
			Name:       repo.FullName,
			Href:       repo.HTMLURL,
			Provider:   p,
		}

		rr = append(rr, r)
	}

	return rr, nil
}

func (g GitHub) Revoke(p *model.Provider) error {
	tok, _ := crypto.Decrypt(p.AccessToken)

	auth := g.Config.ClientID + ":" + g.Config.ClientSecret

	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/applications/%s/tokens/%s", g.APIEndpoint, g.Config.ClientID, string(tok)),
		nil,
	)

	if err != nil {
		return errors.Err(err)
	}

	req.Header.Set("Authorization", "basic " + base64.StdEncoding.EncodeToString([]byte(auth)))

	cli := &http.Client{}

	resp, err := cli.Do(req)

	if err != nil {
		return errors.Err(err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.Err(errors.New("unexpected response from api: " + resp.Status))
	}

	return nil
}

func (g GitHub) Secret() []byte {
	return []byte(g.secret)
}
