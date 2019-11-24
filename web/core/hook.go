package core

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/model/types"
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"
)

type Hook struct {
	web.Handler

	Build           Build
	Providers       model.ProviderStore
	Oauth2Providers map[string]oauth2.Provider
}

type githubUser struct {
	ID    int64
	Login string
}

type githubRepo struct {
	Owner       githubUser
	ContentsURL string `json:"contents_url"`
}

type githubPull struct {
	Number      string
	PullRequest struct {
		Title   string
		URL     string `json:"html_url"`
		Head    struct {
			Sha   string
			Owner githubUser
			Repo  githubRepo `json:"repository"`
		}
	} `json:"pull_request"`
}

type githubPush struct {
	Repo githubRepo `json:"repository"`
	HeadCommit struct {
		ID      string
		URL     string
		Message string
		Author  map[string]string
	} `json:"head_commit"`
}

var githubSignatureLength = 45

func base64DecodeManifest(b []byte) (config.Manifest, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(b))

	if err != nil {
		return config.Manifest{}, errors.Err(err)
	}

	m, err := config.UnmarshalManifest(decoded)

	return m, errors.Err(err)
}

func (h Hook) getGithubManifest(repo githubRepo, ref string) ([]config.Manifest, *model.User, error) {
	mm := make([]config.Manifest, 0)
	u := &model.User{}

	p, err := h.Providers.Get(
		query.Where("provider_user_id", "=", repo.Owner.ID),
		query.Where("name", "=", "github"),
		query.Where("connected", "=", true),
	)

	if err != nil {
		return mm, u, errors.Err(err)
	}

	if err := p.LoadUser(); err != nil {
		return mm, u, errors.Err(err)
	}

	tok, _ := crypto.Decrypt(p.AccessToken)

	rawUrl := strings.Replace(repo.ContentsURL, "{+path}", ".thrall.yml", 1)
	rawUrl += "?ref=" + ref

	url, _ := url.Parse(rawUrl)

	req := &http.Request {
		Method:  "GET",
		URL:     url,
		Header:  http.Header(map[string][]string{
			"Authorization": []string{"token " + string(tok)},
		}),
	}

	cli := &http.Client{}

	resp, err := cli.Do(req)

	if err != nil {
		return mm, u, errors.Err(err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return mm, u, ErrNoManifest
	}

	if resp.StatusCode != http.StatusOK {
		return mm, u, errors.Err(errors.New("Unexpected HTTP status: " + resp.Status))
	}

	file := struct {
		Encoding string
		Content  string
	}{}

	if file.Encoding != "base64" {
		return mm, u, errors.Err(errors.New("Unexpected file encoding: " + file.Encoding))
	}

	m, err := base64DecodeManifest([]byte(file.Content))

	if err != nil {
		return mm, u, ErrBadHookData
	}

	mm = append(mm, m)

	return mm, u, nil
}

func (h Hook) Github(w http.ResponseWriter, r *http.Request) {
	event := r.Header.Get("X-GitHub-Event")
	signature := r.Header.Get("X-Hub-Signature")

	if len(signature) != githubSignatureLength {
		web.Text(w, "Invalid request signature\n", http.StatusForbidden)
		return
	}

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		web.Text(w, err.Error(), http.StatusInternalServerError)
		return
	}

	actual := make([]byte, 20)

	hex.Decode(actual, []byte(signature[5:]))

	expected := hmac.New(sha1.New, h.Oauth2Providers["github"].Secret())
	expected.Write(body)

	if !hmac.Equal(expected.Sum(nil), actual) {
		web.Text(w, "Invalid request signature\n", http.StatusForbidden)
		return
	}

	var (
		mm []config.Manifest
		u  *model.User
		t  *model.Trigger = &model.Trigger{}
	)

	switch event {
	case "ping":
		web.Text(w, "pong\n", http.StatusOK)
		return
	case "push":
		push := &githubPush{}

		json.Unmarshal(body, push)

		mm, u, err = h.getGithubManifest(push.Repo, push.HeadCommit.ID)

		t.Type = types.Push
		t.Comment = push.HeadCommit.Message
		t.Data.Set("provider", "github")
		t.Data.Set("id", push.HeadCommit.ID)
		t.Data.Set("url", push.HeadCommit.URL)
		t.Data.Set("email", push.HeadCommit.Author["email"])
		t.Data.Set("username", push.HeadCommit.Author["username"])
		break
	case "pull_request":
		pull := &githubPull{}

		json.Unmarshal(body, pull)

		mm, u, err = h.getGithubManifest(pull.PullRequest.Head.Repo, pull.PullRequest.Head.Sha)

		t.Type = types.Pull
		t.Comment = pull.PullRequest.Title
		t.Data.Set("provider", "github")
		t.Data.Set("number", pull.Number)
		t.Data.Set("url", pull.PullRequest.URL)
		break
	}

	if err != nil {
		if err == ErrNoManifest {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if err == ErrBadHookData {
			web.Text(w, "Couldn't unmarshal manifest from repository, check validitity", http.StatusBadRequest)
			return
		}

		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		web.Text(w, "Failed to handle hook: " + cause.Error(), http.StatusInternalServerError)
		return
	}

	bb := make([]*model.Build, 0, len(mm))

	for _, m := range mm {
		b, err := h.Build.store(m, u, t)

		if err != nil {
			log.Error.Println(errors.Err(err))

			cause := errors.Cause(err)

			web.Text(w, "Failed to create build: " + cause.Error(), http.StatusInternalServerError)
			break
		}

		bb = append(bb, b)
	}

	for _, b := range bb {
		if err := h.Build.Submit(b, h.Build.Queues[b.Manifest.Driver["type"]]); err != nil {
			log.Error.Println(errors.Err(err))

			cause := errors.Cause(err)

			web.Text(w, "Failed to queue build: " + cause.Error(), http.StatusInternalServerError)
			break
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
