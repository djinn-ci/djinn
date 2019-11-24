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

type githubPush struct {
	Repo struct {
		Owner struct {
			ID int64
		}
		ContentsURL string `json:"contents_url"`
	} `json:"repository"`
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

func (h Hook) handleGithubPush(body []byte) ([]config.Manifest, *model.User, *model.Trigger, error) {
	mm := make([]config.Manifest, 0)
	u := &model.User{}
	t := &model.Trigger{}

	push := &githubPush{}

	json.Unmarshal(body, push)

	p, err := h.Providers.Get(
		query.Where("provider_user_id", "=", push.Repo.Owner.ID),
		query.Where("name", "=", "github"),
		query.Where("connected", "=", true),
	)

	if err != nil {
		return mm, u, t, errors.Err(err)
	}

	if err := p.LoadUser(); err != nil {
		return mm, u, t, errors.Err(err)
	}

	u = p.User

	tok, _ := crypto.Decrypt(p.AccessToken)

	url, _ := url.Parse(strings.Replace(push.Repo.ContentsURL, "{+path}", ".thrall.yml", 1))

	req := &http.Request{
		Method:  "GET",
		URL:     url,
		Header:  http.Header(map[string][]string{
			"Authorization": []string{"token " + string(tok)},
		}),
	}

	cli := &http.Client{}

	resp, err := cli.Do(req)

	if err != nil {
		return mm, u, t, errors.Err(err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return mm, u, t, ErrNoManifest
	}

	if resp.StatusCode != http.StatusOK {
		return mm, u, t, errors.Err(errors.New("Unexpected HTTP status: " + resp.Status))
	}

	file := struct{
		Encoding string
		Content  string
	}{}

	dec := json.NewDecoder(resp.Body)
	dec.Decode(&file)

	if file.Encoding != "base64" {
		return mm, u, t, errors.Err(errors.New("Unexpected file encoding: " + file.Encoding))
	}

	m, err := base64DecodeManifest([]byte(file.Content))

	if err != nil {
		return mm, u, t, ErrBadHookData
	}

	mm = append(mm, m)

	t.Type = types.Push
	t.Comment = push.HeadCommit.Message
	t.Data.Set("provider", "github")
	t.Data.Set("id", push.HeadCommit.ID)
	t.Data.Set("url", push.HeadCommit.URL)
	t.Data.Set("email", push.HeadCommit.Author["email"])
	t.Data.Set("username", push.HeadCommit.Author["username"])

	return mm, u, t, nil
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
		t  *model.Trigger
	)

	switch event {
	case "ping":
		web.Text(w, "pong\n", http.StatusOK)
		return
	case "push":
		mm, u, t, err = h.handleGithubPush(body)
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
