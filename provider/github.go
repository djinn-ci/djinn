package provider

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/andrewpillar/thrall/errors"
)

type githubError struct {
	Message string
	Errors  []map[string]string
}

var (
	githubURL    = "https://api.github.com"
	githubScopes = []string{
		"repo",
		"admin:repo_hook",
	}
)

func unmarshalGitHubRepos(r io.Reader, userId, providerId int64) []*Repo {
	items := make([]struct {
		ID       int64
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	}, 0)

	json.NewDecoder(r).Decode(&items)

	rr := make([]*Repo, 0, len(items))

	for _, it := range items {
		rr = append(rr, &Repo{
			UserID:     userId,
			ProviderID: providerId,
			RepoID:     it.ID,
			Name:       it.FullName,
			Href:       it.HTMLURL,
		})
	}
	return rr
}

func toggleGitHubRepo(cli Client, tok string, r *Repo) error {
	if !r.Enabled {
		body := map[string]interface{}{
			"config": map[string]interface{}{
				"url":          cli.host + "/hook/github",
				"secret":       cli.Secret,
				"content_type": "json",
				"insecure_ssl": 0,
			},
			"events": []string{"push", "pull_request"},
		}

		buf := &bytes.Buffer{}

		json.NewEncoder(buf).Encode(body)

		resp, err := cli.Post(tok, "/repos/" + r.Name + "/hooks", buf)

		if err != nil {
			return errors.Err(err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			err := githubError{}
			json.NewDecoder(resp.Body).Decode(&err)
			return errors.New(err.String())
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

	resp, err := cli.Delete(tok, "/repos/" + r.Name + "/hooks/" + strconv.FormatInt(r.HookID.Int64, 10))

	if err != nil {
		return errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		err := githubError{}
		json.NewDecoder(resp.Body).Decode(&err)
		return errors.New(err.String())
	}

	r.HookID = sql.NullInt64{
		Int64: 0,
		Valid: false,
	}
	r.Enabled = false
	return nil
}

func (e githubError) String() string {
	s := e.Message + ": "

	for i, err := range e.Errors {
		s += err["message"]

		if i != len(e.Errors) - 1 {
			s += ", "
		}
	}
	return s
}
