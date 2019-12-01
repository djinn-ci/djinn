package oauth2

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"

	"golang.org/x/oauth2"
)

type GitHub struct {
	endpoint string
	secret   string

	Config *oauth2.Config
}

var (
	githubScopes = []string{
		"repo",
		"admin:repo_hook",
	}

	githubURL = "https://api.github.com"
)

func (g GitHub) Auth(c context.Context, code string, providers model.ProviderStore) error {
	tok, err := g.Config.Exchange(c, code)

	if err != nil {
		return errors.Err(err)
	}

	p, err := auth(c, "github", tok, providers)

	if err != nil {
		return errors.Err(err)
	}

	resp, err := httpGet("token " + tok.AccessToken, githubURL + "/user")

	if err != nil {
		return errors.Err(err)
	}

	u := struct{
		ID int64
	}{}

	dec := json.NewDecoder(resp.Body)
	dec.Decode(&u)

	p.ProviderUserID = sql.NullInt64{
		Int64: u.ID,
		Valid: true,
	}

	return errors.Err(providers.Update(p))
}

func (g GitHub) AuthURL() string {
	return authURL(g.Config.Endpoint.AuthURL, g.Config.ClientID, githubScopes)
}

func (g GitHub) ToggleRepo(p *model.Provider, id int64) error {
	b, _ := crypto.Decrypt(p.AccessToken)

	tok := "token " + string(b)

	respGet, err := httpGet(tok, fmt.Sprintf("%s/repositories/%v", githubURL, id))

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
				"url":          g.endpoint,
				"secret":       g.secret,
				"content_type": "json",
				"insecure_ssl": 0,
			},
			"events": []string{"push", "pull_request"},
		}

		buf := &bytes.Buffer{}

		enc := json.NewEncoder(buf)
		enc.Encode(body)

		respPost, err := httpPost(
			tok,
			fmt.Sprintf("%s/repos/%s/%s/hooks", githubURL, repo.Owner.Login, repo.Name),
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

	respDelete, err := httpDelete(
		tok,
		fmt.Sprintf("%s/repos/%s/%s/hooks/%v", githubURL, repo.Owner.Login, repo.Name, r.HookID),
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
	b, _ := crypto.Decrypt(p.AccessToken)

	resp, err := httpGet("token " + string(b), githubURL + "/user/repos?sort=updated")

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
	b, _ := crypto.Decrypt(p.AccessToken)

	auth := g.Config.ClientID + ":" + g.Config.ClientSecret

	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/applications/%s/tokens/%s", githubURL, g.Config.ClientID, string(b)),
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
