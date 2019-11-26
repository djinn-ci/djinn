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
	"path/filepath"
	"strconv"
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
	HTMLURL     string     `json:"html_url"`
	ContentsURL string     `json:"contents_url"`
}

type githubPull struct {
	Number      int64
	PullRequest struct {
		Title   string
		URL     string `json:"html_url"`
		Head    struct {
			Sha   string
			Owner githubUser
			Repo  githubRepo
		}
		Base   struct {
			Ref  string
			Repo githubRepo
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

var (
	githubSignatureLength = 45

	invalidManifests = `The hook was accepted, however some of the manifests
retrieved from the repository appeared to contain invalid YAML.`
)

func base64DecodeManifest(b []byte) (config.Manifest, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(b))

	if err != nil {
		return config.Manifest{}, errors.Err(err)
	}

	m, err := config.UnmarshalManifest(decoded)

	return m, errors.Err(err)
}

func githubManifestContents(tok string, rawUrl string) ([]string, error) {
	u, _ := url.Parse(rawUrl)

	req := &http.Request{
		Method: "GET",
		URL:    u,
		Header: http.Header(map[string][]string{
			"Authorization": []string{"token " + tok},
		}),
	}

	cli := &http.Client{}

	resp, err := cli.Do(req)

	if err != nil {
		return []string{}, errors.Err(err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return []string{}, ErrNoManifest
	}

	if resp.StatusCode != http.StatusOK {
		return []string{}, errors.Err(errors.New("Unexpected HTTP status: " + resp.Status))
	}

	type ghFile struct{
		Type     string
		Encoding string
		Content  string
		Links    map[string]string `json:"_links"`
	}

	file := ghFile{}
	dir := make([]ghFile, 0)

	base := filepath.Base(u.Path)

	dec := json.NewDecoder(resp.Body)

	if base == ".thrall.yml" {
		dec.Decode(&file)

		if file.Encoding != "base64" {
			log.Error.Println("Unexpected file encoding from GitHub contents API:", file.Encoding)
			return []string{}, nil
		}

		return []string{file.Content}, nil
	}

	if base == ".thrall" {
		dec.Decode(&dir)

		ss := make([]string, 0, len(dir))

		for _, f := range dir {
			if f.Type == "file" {
				contents, err := githubManifestContents(tok, f.Links["self"])

				if err != nil {
					continue
				}

				ss = append(ss, contents...)
			}
		}

		return ss, nil
	}

	return []string{}, nil
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

	u = p.User

	tok, _ := crypto.Decrypt(p.AccessToken)

	fileUrl := strings.Replace(repo.ContentsURL, "{+path}", ".thrall.yml", 1)
	fileUrl += "?ref=" + ref

	dirUrl := strings.Replace(repo.ContentsURL, "{+path}", ".thrall", 1)
	dirUrl += "?ref=" + ref

	b64Manifests := make([]string, 0)

	for _, url := range []string{fileUrl, dirUrl} {
		contents, err := githubManifestContents(string(tok), url)

		if err != nil {
			if err == ErrNoManifest {
				continue
			}

			return mm, u, errors.Err(err)
		}

		b64Manifests = append(b64Manifests, contents...)
	}

	for _, b64Manifest := range b64Manifests {
		var m config.Manifest

		m, err = base64DecodeManifest([]byte(b64Manifest))

		if err == nil {
			mm = append(mm, m)
			continue
		}

		// Decoding will only fail if the encoded base64 string is not a valid
		// YAML document.
		err = ErrBadHookData
	}

	return mm, u, err
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
		t.Data.Set("number", strconv.FormatInt(pull.Number, 10))
		t.Data.Set("url", pull.PullRequest.URL)
		t.Data.Set("ref", pull.PullRequest.Base.Ref)
		t.Data.Set("ref_url", pull.PullRequest.Base.Repo.HTMLURL)
		break
	}

	if err != nil && err != ErrInvalidManifest {
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

			if cause == ErrNamespaceNameInvalid {
				web.Text(w, "Failed to create build: namespace name can only contain letters or numbers", http.StatusBadRequest)
				return
			}

			web.Text(w, "Failed to create build: " + cause.Error(), http.StatusInternalServerError)
			return
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

	// We failed to decode some of the manifests, as long as we created some
	// builds this is ok. Write a response back saying that some of the
	// manifests were invalid.
	if err == ErrBadHookData {
		web.Text(w, invalidManifests, http.StatusAccepted)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}
