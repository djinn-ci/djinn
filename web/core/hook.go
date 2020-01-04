package core

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
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

type manifestDecoder func(io.Reader) (config.Manifest, error)

type manifestErrors map[string]error

type Hook struct {
	web.Handler

	Build           Build
	Providers       model.ProviderStore
	Oauth2Providers map[string]oauth2.Provider
}

var (
	githubSignatureLength = 45

	invalidManifest = `The hook was accepted, however some of the manifests
retrieved from the repository appeared to contain invalid YAML. See below:`

	invalidNamespaceName = `Invalid namespace name in build manifest, the name
can only contain letters or numbers`
)

func decodeBase64JsonManifest(r io.Reader) (config.Manifest, error) {
	file := struct{
		Encoding string
		Content  string
	}{}

	dec := json.NewDecoder(r)

	if err := dec.Decode(&file); err != nil {
		return config.Manifest{}, errors.Err(err)
	}

	if file.Encoding != "base64" {
		return config.Manifest{}, errors.Err(errors.New("unexpected file encoding: " + file.Encoding))
	}

	raw, err := base64.StdEncoding.DecodeString(file.Content)

	if err != nil {
		return config.Manifest{}, errors.Err(err)
	}

	m, err := config.UnmarshalManifest([]byte(raw))

	return m, err
}

func (e manifestErrors) Error() string {
	buf := &bytes.Buffer{}

	for file, err := range e {
		buf.WriteString(file + ": " + err.Error() + "\n")
	}

	return buf.String()
}

func (h Hook) getToken(name string, userId int64) (string, error) {
	p, err := h.Providers.Get(
		query.Where("provider_user_id", "=", userId),
		query.Where("name", "=", name),
		query.Where("connected", "=", true),
	)

	if err != nil {
		return "", errors.Err(err)
	}

	tok, _ := crypto.Decrypt(p.AccessToken)

	return string(tok), nil
}

func (h Hook) getUserFromProvider(name string, userId int64) (*model.User, error) {
	u, err := h.Users.Get(
		query.WhereQuery("id", "=",
			query.Select(
				query.Columns("user_id"),
				query.From("providers"),
				query.Where("provider_user_id", "=", userId),
				query.Where("name", "=", name),
				query.Where("connected", "=", true),
			),
		),
	)

	return u, errors.Err(err)
}

func (h Hook) loadManifests(fn manifestDecoder, tok string, urls []string) ([]config.Manifest, error) {
	errs := manifestErrors(make(map[string]error))
	mm := make([]config.Manifest, 0, len(urls))

	cli := &http.Client{}

	for _, raw := range urls {
		u, _ := url.Parse(raw)

		req := &http.Request{
			Method: "GET",
			URL:    u,
			Header: http.Header(map[string][]string{
				"Authorization": []string{"Bearer " + tok},
			}),
		}

		resp, err := cli.Do(req)

		if err != nil {
			log.Error.Println(errors.Err(err))
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Error.Println(errors.Err(errors.New("unexpected http status for " + raw + ": " + resp.Status)))
			continue
		}

		m, err := fn(resp.Body)

		if err != nil {
			parts := strings.Split(strings.Split(raw, "?")[0], "/")

			errs[strings.Join(parts[len(parts)-2:], "/")] = err
			continue
		}

		mm = append(mm, m)
	}

	if len(errs) > 0 {
		return mm, errs
	}

	return mm, nil
}

func (h Hook) getGithubManifestUrls(tok, rawUrl, ref string) ([]string, error) {
	urls := []string{
		strings.Replace(rawUrl, "{+path}", ".thrall.yml", 1) + "?ref=" + ref,
	}

	u, _ := url.Parse(strings.Replace(rawUrl, "{+path}", ".thrall", 1) + "?ref=" + ref)

	cli := &http.Client{}

	req := &http.Request{
		Method: "GET",
		URL:    u,
		Header: http.Header(map[string][]string{
			"Authorization": []string{"Bearer " + tok},
		}),
	}

	resp, err := cli.Do(req)

	if err != nil {
		return []string{}, errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return urls, nil
	}

	if resp.StatusCode != http.StatusOK {
		err := errors.New("unexpected http status for " + u.String() + ": " + resp.Status)

		return []string{},  errors.Err(err)
	}

	items := make([]struct{
		Type string
		Name string
		URL  string
	}, 0)

	dec := json.NewDecoder(resp.Body)
	dec.Decode(&items)

	for _, it := range items {
		if it.Type == "dir" {
			continue
		}

		if strings.HasSuffix(it.Name, ".yml") || strings.HasSuffix(it.Name, ".yaml") {
			urls = append(urls, it.URL)
		}
	}

	return urls, nil
}

func (h Hook) getGitlabManifestUrls(tok, rawUrl, ref string) ([]string, error) {
	urls := []string{
		rawUrl + "/repository/files/.thrall.yml?ref=" + ref,
	}

	u, _ := url.Parse(rawUrl + "/repository/tree?path=.thrall&?ref=" + ref)

	cli := &http.Client{}

	req := &http.Request{
		Method: "GET",
		URL:    u,
		Header: http.Header(map[string][]string{
			"Authorization": []string{"Bearer " + tok},
		}),
	}

	resp, err := cli.Do(req)

	if err != nil {
		return []string{}, errors.Err(err)
	}

	defer resp.Body.Close()

	items := make([]struct{
		Type string
		Path string
	}, 0)

	dec := json.NewDecoder(resp.Body)
	dec.Decode(&items)

	for _, it := range items {
		if it.Type == "tree" {
			continue
		}

		urls = append(urls, rawUrl + "/reposoitory/files/" + url.QueryEscape(it.Path) + "?ref=" + ref)
	}

	return urls, nil
}

func (h Hook) submitBuilds(mm []config.Manifest, u *model.User, t *model.Trigger) error {
	bb := make([]*model.Build, 0, len(mm))

	for _, m := range mm {
		b, err := h.Build.store(m, u, t)

		if err != nil {
			return errors.Err(err)
		}

		bb = append(bb, b)
	}

	for _, b := range bb {
		if err := h.Build.Submit(b, h.Build.Queues[b.Manifest.Driver["type"]]); err != nil {
			return errors.Err(err)
		}
	}

	return nil
}

func (h Hook) Github(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

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

	provider := h.Oauth2Providers["github"]

	expected := hmac.New(sha1.New, provider.Secret())
	expected.Write(body)

	if !hmac.Equal(expected.Sum(nil), actual) {
		web.Text(w, "Invalid request signature\n", http.StatusForbidden)
		return
	}

	var (
		loadErr error
		mm      []config.Manifest
		u       *model.User
		t       *model.Trigger = &model.Trigger{}

		actions = map[string]struct{}{
			"opened":      {},
			"reopened":    {},
			"unlocked":    {},
			"synchronize": {},
		}
	)

	type user struct {
		ID    int64
		Login string
	}

	type repo struct {
		Owner       user
		HTMLURL     string `json:"html_url"`
		ContentsURL string `json:"contents_url"`
	}

	type pushData struct {
		Ref        string
		Repo       repo `json:"repository"`
		HeadCommit struct{
			ID      string
			URL     string
			Message string
			Author  map[string]string
		} `json:"head_commit"`
	}

	type pullData struct {
		Action      string
		Number      int64
		PullRequest struct{
			Title string
			User  user
			URL   string `json:"html_url"`
			Head  struct{
				Sha   string
				Owner user
				Repo  repo
			}
			Base  struct{
				Ref  string
				Repo repo
			}
		} `json:"pull_request"`
	}

	switch event {
	case "ping":
		web.Text(w, "pong\n", http.StatusOK)
		return
	case "push":
		push := &pushData{}

		json.Unmarshal(body, push)

		u_, err := h.getUserFromProvider("github", push.Repo.Owner.ID)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		u = u_

		tok, err := h.getToken("github", push.Repo.Owner.ID)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		urls, err := h.getGithubManifestUrls(tok, push.Repo.ContentsURL, push.HeadCommit.ID)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		mm, loadErr = h.loadManifests(decodeBase64JsonManifest, tok, urls)

		t.Type = types.Push
		t.Comment = push.HeadCommit.Message
		t.Data.Set("id", push.HeadCommit.ID)
		t.Data.Set("url", push.HeadCommit.URL)
		t.Data.Set("ref", push.Ref)
		t.Data.Set("email", push.HeadCommit.Author["email"])
		t.Data.Set("username", push.HeadCommit.Author["username"])
		break
	case "pull_request":
		pull := &pullData{}

		json.Unmarshal(body, pull)

		if _, ok := actions[pull.Action]; !ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		u_, err := h.getUserFromProvider("github", pull.PullRequest.User.ID)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		u = u_

		tok, err := h.getToken("github", pull.PullRequest.User.ID)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		urls, err := h.getGithubManifestUrls(
			tok,
			pull.PullRequest.Head.Repo.ContentsURL,
			pull.PullRequest.Head.Sha,
		)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		mm, loadErr = h.loadManifests(decodeBase64JsonManifest, tok, urls)

		action := pull.Action

		if action == "synchronize" {
			action = "synchronized"
		}

		t.Type = types.Pull
		t.Comment = pull.PullRequest.Title
		t.Data.Set("id", strconv.FormatInt(pull.Number, 10))
		t.Data.Set("url", pull.PullRequest.URL)
		t.Data.Set("ref", pull.PullRequest.Base.Ref)
		t.Data.Set("username", pull.PullRequest.User.Login)
		t.Data.Set("action", action)
		break
	}

	if err := h.submitBuilds(mm, u, t); err != nil {
		cause := errors.Cause(err)

		if cause == ErrNamespaceNameInvalid {
			web.Text(w, invalidNamespaceName, http.StatusBadRequest)
			return
		}

		web.Text(w, "Failed to create build: " + cause.Error(), http.StatusInternalServerError)
		return
	}

	if loadErr != nil {
		web.Text(w, invalidManifest + "\n\n" + loadErr.Error(), http.StatusAccepted)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h Hook) Gitlab(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	event := r.Header.Get("X-Gitlab-Event")
	token := r.Header.Get("X-Gitlab-Token")

	if !bytes.Equal([]byte(token), h.Oauth2Providers["gitlab"].Secret()) {
		web.Text(w, "Invalid request token\n", http.StatusForbidden)
		return
	}

	var (
		err error
		mm  []config.Manifest
		u   *model.User
		t   *model.Trigger = &model.Trigger{}
	)

	type repo struct {
		ID     int64
		URL    string
		WebURL string `json:"web_url"`
	}

	type pushData struct {
		After   string
		Ref     string
		UserID  int64 `json:"user_id"`
		Project repo
		Commits []struct{
			ID      string
			Message string
			URL     string
			Author  map[string]string
		}
	}

	type mergeData struct {
		User  struct{
			Name string
		}
		Attrs struct{
			ID              int64
			SourceBranch    string `json:"source_branch"`
			SourceProjectID int64  `json:"source_project_id"`
			AuthorID        int64  `json:"author_id"`
			Title           string
			LastCommit      struct{
				ID string
			} `json:"last_commit"`
			URL    string
			Action string
		} `json:"object_attributes"`
	}

	switch event {
	case "Push Hook":
		push := &pushData{}

		dec := json.NewDecoder(r.Body)
		dec.Decode(push)

		u, err = h.getUserFromProvider("gitlab", push.UserID)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		tok, err := h.getToken("gitlab", push.UserID)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		headCommit := push.Commits[len(push.Commits) - 1]

		webUrl, _ := url.Parse(push.Project.WebURL)

		urls, err := h.getGitlabManifestUrls(
			tok,
			fmt.Sprintf("%s/api/v4/projects/%v", webUrl.Scheme+"://"+webUrl.Host, push.Project.ID),
			headCommit.ID,
		)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		mm, err = h.loadManifests(decodeBase64JsonManifest, tok, urls)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		t.Type = types.Push
		t.Comment = headCommit.Message
		t.Data.Set("id", headCommit.ID)
		t.Data.Set("url", headCommit.URL)
		t.Data.Set("ref", push.Ref)
		t.Data.Set("email", headCommit.Author["email"])
		t.Data.Set("username", headCommit.Author["username"])
		break
	case "Merge Request Hook":
		merge := &mergeData{}

		dec := json.NewDecoder(r.Body)
		dec.Decode(merge)

		u, err = h.getUserFromProvider("gitlab", merge.Attrs.AuthorID)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		tok, err := h.getToken("gitlab", merge.Attrs.AuthorID)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		urls, err := h.getGitlabManifestUrls(
			tok,
			fmt.Sprintf("%s/api/v4/projects/%v", r.Header.Get("Referer"), merge.Attrs.SourceProjectID),
			merge.Attrs.LastCommit.ID,
		)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		mm, err = h.loadManifests(decodeBase64JsonManifest, tok, urls)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		t.Type = types.Pull
		t.Comment = merge.Attrs.Title
		t.Data.Set("id", strconv.FormatInt(merge.Attrs.ID, 10))
		t.Data.Set("url", merge.Attrs.URL)
		t.Data.Set("ref", merge.Attrs.SourceBranch)
		t.Data.Set("username", merge.User.Name)
		t.Data.Set("action", merge.Attrs.Action)
		return
	}

	if err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		web.Text(w, "Failed to handle hook: " + cause.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.submitBuilds(mm, u, t); err != nil {
		cause := errors.Cause(err)

		if cause == ErrNamespaceNameInvalid {
			web.Text(w, invalidNamespaceName, http.StatusBadRequest)
			return
		}

		web.Text(w, "Failed to create build: " + cause.Error(), http.StatusInternalServerError)
		return
	}

	if _, ok := err.(manifestErrors); ok {
		web.Text(w, invalidManifest + "\n\n" + err.Error(), http.StatusAccepted)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
