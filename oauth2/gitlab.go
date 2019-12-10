package oauth2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"
)

type GitLab struct {
	Client
}

var (
	gitlabScopes = []string{"api"}

	gitlabURL = "https://gitlab.com"
)

func (g GitLab) Auth(c context.Context, code string, providers model.ProviderStore) error {
	return errors.Err(g.auth(c, "gitlab", code, providers))
}

func (g GitLab) ToggleRepo(p *model.Provider, id int64) error {
	tok, _ := crypto.Decrypt(p.AccessToken)

	r, err := p.RepoStore().Get(query.Where("repo_id", "=", id))

	if err != nil {
		return errors.Err(err)
	}

	if !r.Enabled {
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
		err = enc.Encode(body)

		if err != nil {
			println(err.Error())
		}

		resp, err := g.Post(string(tok), fmt.Sprintf("%s/projects/%v/hooks", g.Endpoint, id), buf)

		if err != nil {
			return errors.Err(err)
		}

		if resp.StatusCode != http.StatusCreated {
			return errors.Err(errors.New("unexpected http status: " + resp.Status))
		}

		defer resp.Body.Close()

		hook := struct{
			ID int64
		}{}

		dec := json.NewDecoder(resp.Body)
		dec.Decode(&hook)

		return errors.Err(toggleRepo(p, id, hook.ID))
	}

	resp, err := g.Delete(string(tok), fmt.Sprintf("%s/projects/%v/hooks/%v", g.Endpoint, id, r.HookID))

	if err != nil {
		return errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return errors.Err(errors.New("failed to toggle repository hook: " + resp.Status))
	}

	return errors.Err(toggleRepo(p, id, r.HookID))
}

func (g GitLab) Repos(p *model.Provider) ([]*model.Repo, error) {
	tok, _ := crypto.Decrypt(p.AccessToken)

	url := fmt.Sprintf("%s/users/%v/projects?simple=true&order_by=updated_at", g.Endpoint, p.ProviderUserID.Int64)

	resp, err := g.Get(string(tok), url)

	if err != nil {
		return []*model.Repo{}, errors.Err(err)
	}

	defer resp.Body.Close()

	projects := make([]struct{
		ID     int64
		Name   string `json:"path_with_namespace"`
		WebURL string `json:"web_url"`
	}, 0)

	dec := json.NewDecoder(resp.Body)
	dec.Decode(&projects)

	rr := make([]*model.Repo, 0, len(projects))

	for _, proj := range projects {
		r := &model.Repo{
			ProviderID: p.ID,
			RepoID:     proj.ID,
			Name:       proj.Name,
			Href:       proj.WebURL,
			Provider:   p,
		}

		rr = append(rr, r)
	}

	return rr, nil
}

func (g GitLab) Revoke(p *model.Provider) error {
	return nil
}
