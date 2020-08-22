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

var (
	gitlabURL    = "https://gitlab.com/api/v4"
	gitlabScopes = []string{"api"}
)

func unmarshalGitLabRepos(r io.Reader, userId int64, providerId int64) []*Repo {
	items := make([]struct {
		ID     int64
		Name   string `json:"path_with_namespace"`
		WebURL string `json:"web_url"`
	}, 0)

	json.NewDecoder(r).Decode(&items)

	rr := make([]*Repo, 0, len(items))

	for _, it := range items {
		rr = append(rr, &Repo{
			UserID:     userId,
			ProviderID: providerId,
			RepoID:     it.ID,
			Name:       it.Name,
			Href:       it.WebURL,
		})
	}
	return rr
}

func toggleGitLabRepo(cli Client, tok string, r *Repo) error {
	if !r.Enabled {
		body := map[string]interface{}{
			"id":                      r.RepoID,
			"url":                     cli.host + "/hook/gitlab",
			"push_events":             true,
			"merge_requests_events":   true,
			"enable_ssl_verification": true,
			"token":                   cli.Secret,
		}

		buf := &bytes.Buffer{}

		json.NewEncoder(buf).Encode(body)

		resp, err := cli.Post(tok, "/projects/" + strconv.FormatInt(r.RepoID, 10), buf)

		if err != nil {
			return errors.Err(err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			return errors.New("unexpected http status " + resp.Status)
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

	resp, err := cli.Delete(tok, "/projects/" + strconv.FormatInt(r.RepoID, 10) + "/hooks/" + strconv.FormatInt(r.HookID.Int64, 10))

	if err != nil {
		return errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return errors.New("unexpected http status " + resp.Status)
	}

	r.HookID = sql.NullInt64{
		Int64: 0,
		Valid: false,
	}
	r.Enabled = false
	return nil
}
