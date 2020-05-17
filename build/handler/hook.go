package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"database/sql"
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
	"sync"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/provider"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"
)

var (
	githubSignatureLength = 45

	invalidManifest = `The hook was accepted, however some of the manifests retrieved from the
repository appeared to contain invalid YAML, see below:`

	invalidNamespaceName = `Invalid namespace name in build manifest, the name can only contain letters or
numbers`
)

type Hook struct {
	Build
}

type manifestDecoder func(io.Reader) (config.Manifest, error)

func providerUserId(name string, id int64) query.Query {
	return provider.Select(
		"user_id",
		query.Where("provider_user_id", "=", id),
		query.Where("name", "=", name),
		query.Where("connected", "=", true),
	)
}

func decodeBase64JSONManifest(r io.Reader) (config.Manifest, error) {
	file := struct{
		Encoding string
		Content  string
	}{}

	if err := json.NewDecoder(r).Decode(&file); err != nil {
		return config.Manifest{}, errors.Err(err)
	}

	if file.Encoding != "base64" {
		return config.Manifest{}, errors.New("unexpected file encoding: "+file.Encoding)
	}

	raw, err := base64.StdEncoding.DecodeString(file.Content)

	if err != nil {
		return config.Manifest{}, errors.Err(err)
	}

	m, err := config.UnmarshalManifest([]byte(raw))
	return m, errors.Err(err)
}

func (h Hook) loadManifests(decode manifestDecoder, tok string, urls []string) ([]config.Manifest, error) {
	errs := make(chan error)
	manifests := make(chan config.Manifest)

	wg := &sync.WaitGroup{}

	for _, raw := range urls {
		wg.Add(1)

		go func(wg *sync.WaitGroup, raw string) {
			defer wg.Done()

			url, _ := url.Parse(raw)

			cli := &http.Client{}
			resp, err := cli.Do(&http.Request{
				Method: "GET",
				URL:    url,
				Header: http.Header(map[string][]string{
					"Authorization": []string{"Bearer "+tok},
				}),
			})

			if err != nil {
				errs <- err
				return
			}

			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				return
			}

			if resp.StatusCode != http.StatusOK {
				errs <- errors.New("unexpected http status for "+url.String()+": "+resp.Status)
				return
			}

			m, err := decode(resp.Body)

			if err != nil {
				parts := strings.Split(strings.Split(raw, "?")[0], "/")
				path := strings.Join(parts[len(parts)-2:], "/")

				errs <- errors.New("failed to decode manifest "+path+":\n"+errors.Cause(err).Error())
				return
			}
			manifests <- m
		}(wg, raw)
	}

	go func() {
		wg.Wait()
		close(errs)
		close(manifests)
	}()

	mm := make([]config.Manifest, 0, len(urls))
	ee := make([]error, 0, len(urls))

	for errs != nil && manifests != nil {
		select {
		case err, ok := <-errs:
			if !ok {
				errs = nil
				break
			}
			ee = append(ee, err)
		case m, ok := <-manifests:
			if !ok {
				errs = nil
				break
			}
			mm = append(mm, m)
		}
	}

	if len(ee) > 0 {
		return mm, errors.MultiError(ee...)
	}
	return mm, nil
}

func (h Hook) getGithubManifestUrls(tok, rawUrl, ref string) ([]string, error) {
	urls := []string{strings.Replace(rawUrl, "{+path}", ".thrall.yml", 1) + "?ref="+ref}

	url, _ := url.Parse(strings.Replace(rawUrl, "{+path}", ".thrall", 1)+"?ref="+ref)

	cli := &http.Client{}

	resp, err := cli.Do(&http.Request{
		Method: "GET",
		URL:    url,
		Header: http.Header(map[string][]string{
			"Authorization": []string{"Bearer "+tok},
		}),
	})

	if err != nil {
		return []string{}, errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return urls, nil
	}

	if resp.StatusCode != http.StatusOK {
		return []string{}, errors.New("unexpected http status for "+url.String()+": "+resp.Status)
	}

	items := make([]struct{
		Type string
		Name string
		URL  string
	}, 0)

	json.NewDecoder(resp.Body).Decode(&items)

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
	urls := []string{rawUrl+"/repository/files/thrall.yml?ref="+ref}

	parsed, _ := url.Parse(rawUrl+"/repository/tree?path=.thrall&ref="+ref)

	cli := &http.Client{}

	resp, err := cli.Do(&http.Request{
		Method: "GET",
		URL:    parsed,
		Header: http.Header(map[string][]string{
			"Authorization": []string{"Bearer "+tok},
		}),
	})

	if err != nil {
		return []string{}, errors.Err(err)
	}

	defer resp.Body.Close()

	items := make([]struct{
		Type string
		Path string
	}, 0)

	json.NewDecoder(resp.Body).Decode(&items)

	for _, it := range items {
		if it.Type == "tree" {
			continue
		}
		urls = append(urls, rawUrl+"/repository/files/"+url.QueryEscape(it.Path)+"?ref="+ref)
	}
	return urls, nil
}

func (h Hook) getProvider(name string, userId int64) (*provider.Provider, error) {
	p, err := h.Providers.Get(
		query.Where("provider_user_id", "=", userId),
		query.Where("name", "=", name),
		query.Where("connected", "=", true),
	)
	return p, errors.Err(err)
}

func (h Hook) submitBuilds(mm []config.Manifest, u *user.User, t *build.Trigger) error {
	bb := make([]*build.Build, 0, len(mm))

	for _, m := range mm {
		b, err := h.realStore(m, u, t)

		if err != nil {
			return errors.Err(err)
		}
		bb = append(bb, b)
	}

	for _, b := range bb {
		if err := h.Submit(b, h.Queues[b.Manifest.Driver["type"]]); err != nil {
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
		log.Error.Println(r.Method, r.URL, errors.Err(err))
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
		manifestErr error
		mm          []config.Manifest
		u           *user.User
		t           *build.Trigger = &build.Trigger{}

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
			Head struct{
				Sha   string
				Owner user
				Repo  repo
			}
			Base struct{
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

		var err error

		u, err = h.Users.Get(query.WhereQuery("id", "=", providerUserId("github", push.Repo.Owner.ID)))

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		p, err := h.getProvider("github", push.Repo.Owner.ID)

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		b, _ := crypto.Decrypt(p.AccessToken)
		tok := string(b)

		urls, err := h.getGithubManifestUrls(tok, push.Repo.ContentsURL, push.HeadCommit.ID)

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		mm, manifestErr = h.loadManifests(decodeBase64JSONManifest, tok, urls)

		t.ProviderID = sql.NullInt64{
			Int64: p.ID,
			Valid: true,
		}
		t.Type = build.Push
		t.Comment = push.HeadCommit.Message
		t.Data.Set("id", push.HeadCommit.ID)
		t.Data.Set("url", push.HeadCommit.URL)
		t.Data.Set("ref", push.Ref)
		t.Data.Set("email", push.HeadCommit.Author["email"])
		t.Data.Set("username", push.HeadCommit.Author["username"])
	case "pull_request":
		pull := &pullData{}

		json.Unmarshal(body, pull)

		var err error

		if _, ok := actions[pull.Action]; !ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		u, err = h.Users.Get(query.WhereQuery("id", "=", providerUserId("github", pull.PullRequest.User.ID)))

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		p, err := h.getProvider("github", pull.PullRequest.User.ID)

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		b, _ := crypto.Decrypt(p.AccessToken)
		tok := string(b)

		urls, err := h.getGithubManifestUrls(tok, pull.PullRequest.Head.Repo.ContentsURL, pull.PullRequest.Head.Sha)

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		mm, manifestErr = h.loadManifests(decodeBase64JSONManifest, tok, urls)

		action := pull.Action

		if action == "synchronize" {
			action = "synchronized"
		}

		t.ProviderID = sql.NullInt64{
			Int64: p.ID,
			Valid: true,
		}
		t.Type = build.Pull
		t.Comment = pull.PullRequest.Title
		t.Data.Set("id", strconv.FormatInt(pull.Number, 10))
		t.Data.Set("url", pull.PullRequest.URL)
		t.Data.Set("ref", pull.PullRequest.Base.Ref)
		t.Data.Set("username", pull.PullRequest.User.Login)
		t.Data.Set("action", action)
	}

	if err := h.submitBuilds(mm, u, t); err != nil {
		cause := errors.Cause(err)

		if cause == namespace.ErrName {
			web.Text(w, invalidNamespaceName, http.StatusBadRequest)
			return
		}
		log.Error.Println(errors.Err(err))
		web.Text(w, "Failed to create build: "+cause.Error(), http.StatusInternalServerError)
		return
	}

	if manifestErr != nil {
		web.Text(w, invalidManifest+"\n\n"+manifestErr.Error(), http.StatusAccepted)
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
		manifestErr error
		mm          []config.Manifest
		u           *user.User
		t           *build.Trigger = &build.Trigger{}
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
		User    struct{
			Name string
		}
		Project repo
		Attrs   struct{
			ID              int64  `json:"iid"`
			TargetBranch    string `json:"target_branch"`
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

		json.NewDecoder(r.Body).Decode(push)

		var err error

		u, err = h.Users.Get(query.WhereQuery("id", "=", providerUserId("github", push.UserID)))

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		p, err := h.getProvider("gitlab", push.UserID)

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		b, _ := crypto.Decrypt(p.AccessToken)
		tok := string(b)

		head := push.Commits[len(push.Commits)-1]
		url, _ := url.Parse(push.Project.WebURL)

		urls, err := h.getGitlabManifestUrls(tok, fmt.Sprintf("%s/api/v4/projects/%v", url.String(), push.Project.ID), head.ID)

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		mm, manifestErr = h.loadManifests(decodeBase64JSONManifest, tok, urls)

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		t.Type = build.Push
		t.Comment = head.Message
		t.Data.Set("id", head.ID)
		t.Data.Set("url", head.URL)
		t.Data.Set("ref", push.Ref)
		t.Data.Set("email", head.Author["email"])
		t.Data.Set("username", head.Author["username"])
	case "Merge Request Hook":
		merge := &mergeData{}

		json.NewDecoder(r.Body).Decode(merge)

		var err error

		u, err = h.Users.Get(query.WhereQuery("id", "=", providerUserId("github", merge.Attrs.AuthorID)))

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		p, err := h.getProvider("gitlab", merge.Attrs.AuthorID)

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		b, _ := crypto.Decrypt(p.AccessToken)
		tok := string(b)

		url, _ := url.Parse(merge.Project.WebURL)

		urls, err := h.getGitlabManifestUrls(tok, fmt.Sprintf("%s/api/v4/projects/%v", url.String(), merge.Attrs.LastCommit.ID), merge.Attrs.LastCommit.ID)

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		mm, manifestErr = h.loadManifests(decodeBase64JSONManifest, tok, urls)

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		t.Type = build.Pull
		t.Comment = merge.Attrs.Title
		t.Data.Set("id", strconv.FormatInt(merge.Attrs.ID, 10))
		t.Data.Set("url", merge.Attrs.URL)
		t.Data.Set("ref", merge.Attrs.TargetBranch)
		t.Data.Set("username", merge.User.Name)
		t.Data.Set("action", merge.Attrs.Action)
	}

	if err := h.submitBuilds(mm, u, t); err != nil {
		cause := errors.Cause(err)

		if cause == namespace.ErrName {
			web.Text(w, invalidNamespaceName, http.StatusBadRequest)
			return
		}
		log.Error.Println(errors.Err(err))
		web.Text(w, "Failed to create build: "+cause.Error(), http.StatusInternalServerError)
		return
	}

	if manifestErr != nil {
		web.Text(w, invalidManifest+"\n\n"+manifestErr.Error(), http.StatusAccepted)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
