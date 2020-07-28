package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/oauth2"
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

	Providers       *provider.Store
	Oauth2Providers map[string]oauth2.Provider
}

type manifestDecoder func(io.Reader) (config.Manifest, error)

func (h Hook) getUserAndToken(name string, userId int64) (*user.User, int64, string, error) {
	u, err := h.Users.Get(query.WhereQuery("id", "=",
		provider.Select(
			"user_id",
			query.Where("provider_user_id", "=", userId),
			query.Where("name", "=", name),
			query.Where("connected", "=", true),
		),
	))

	if err != nil {
		return u, 0, "", errors.Err(err)
	}

	p, err := h.Providers.Get(
		query.Where("provider_user_id", "=", userId),
		query.Where("name", "=", name),
		query.Where("connected", "=", true),
	)

	if err != nil {
		return u, 0, "", errors.Err(err)
	}

	b, _ := h.Block.Decrypt(p.AccessToken)
	return u, p.ID, string(b), nil
}

func decodeBase64JSONManifest(r io.Reader) (config.Manifest, error) {
	file := struct {
		Encoding string
		Content  string
	}{}

	if err := json.NewDecoder(r).Decode(&file); err != nil {
		return config.Manifest{}, errors.Err(err)
	}

	if file.Encoding != "base64" {
		return config.Manifest{}, errors.New("unexpected file encoding: " + file.Encoding)
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
					"Authorization": []string{"Bearer " + tok},
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
				errs <- errors.New("unexpected http status for " + url.String() + ": " + resp.Status)
				return
			}

			m, err := decode(resp.Body)

			if err != nil {
				parts := strings.Split(strings.Split(raw, "?")[0], "/")
				path := strings.Join(parts[len(parts)-2:], "/")

				errs <- errors.New("failed to decode manifest " + path + ":\n" + errors.Cause(err).Error())
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

func (h Hook) getManifestURLs(tok, rawurl string, geturl func(map[string]string) string) ([]string, error) {
	urls := make([]string, 0)

	url, _ := url.Parse(rawurl)

	cli := &http.Client{}

	resp, err := cli.Do(&http.Request{
		Method: "GET",
		URL:    url,
		Header: http.Header(map[string][]string{
			"Authorization": []string{"Bearer " + tok},
		}),
	})

	if err != nil {
		return urls, errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		h.Log.Debug.Println("no manifest urls found")
		return urls, nil
	}

	items := make([]map[string]string, 0)

	json.NewDecoder(resp.Body).Decode(&items)

	for _, it := range items {
		if rawurl := geturl(it); rawurl != "" {
			urls = append(urls, rawurl)
		}
	}
	return urls, nil
}

// submitBuilds will create and submit a build for each manifest in the given
// slice of manifests.
func (h Hook) submitBuilds(mm []config.Manifest, u *user.User, t *build.Trigger) error {
	bb := make([]*build.Build, 0, len(mm))

	for _, m := range mm {
		b, err := build.NewStore(h.DB, u).Create(m, t)

		if err != nil {
			return errors.Err(err)
		}
		bb = append(bb, b)
	}

	for _, b := range bb {
		if err := build.NewStoreWithHasher(h.DB, h.Hasher).Submit(h.Queues[b.Manifest.Driver["type"]], b); err != nil {
			return errors.Err(err)
		}
	}
	return nil
}

// Github is the webhook for the ping, push, and pull_request events. Before
// each event is handled individually the request's X-Hub-Signature is checked
// to make sure it is valid. On a successful handling of an event this sends
// back a 204 No Content, otherwise we send back a plain text response with
// 500 Internal Server Error, and the verbatim error cause in the response
// body. Detailed below is how each event is handled,
//
// ping - This simple sends a plain text response back with 200 OK and "pong\n"
// as the response body.
//
// push - This will extract information about the author of the pushed commit
// and use this to create a build.Push build.Trigger. The data of the trigger
// will contain the id of the head commit, the url to the head commit, the ref
// of the pushed commit, along with the email and username from the pushed
// commit. The build manifests are then extracted from the repository that
// was pushed to.
//
// pull_request - This will create a build.Pull build.Trigger. The data of the
// trigger will contain the id of the pull request, the url of the pull request
// and the ref of the pull request. The username of the pull request author is
// included, but not the email. The action key in the build data is set to
// whatever the action is on the incoming pull request event, unless the
// action is synchronize, in which case this is changed to synchronized. The
// build manifests are then extracted from the head repository of the pull
// request.
func (h Hook) Github(w http.ResponseWriter, r *http.Request) {
	h.Log.Debug.Println("github webhook received")

	event := r.Header.Get("X-GitHub-Event")
	signature := r.Header.Get("X-Hub-Signature")

	if len(signature) != githubSignatureLength {
		web.Text(w, "Invalid request signature\n", http.StatusForbidden)
		return
	}

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
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
		providerId  int64
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
		HeadCommit struct {
			ID      string
			URL     string
			Message string
			Author  map[string]string
		} `json:"head_commit"`
	}

	type pullData struct {
		Action      string
		Number      int64
		PullRequest struct {
			Title string
			User  user
			URL   string `json:"html_url"`
			Head  struct {
				Sha   string
				Owner user
				Repo  repo
			}
			Base struct {
				Ref  string
				Repo repo
			}
		} `json:"pull_request"`
	}

	h.Log.Debug.Println("github event type:", event)

	geturl := func(m map[string]string) string {
		if m["type"] == "dir" {
			return ""
		}

		if strings.HasSuffix(m["name"], ".yml") || strings.HasSuffix(m["name"], ".yaml") {
			return m["url"]
		}
		return ""
	}

	switch event {
	case "ping":
		web.Text(w, "pong\n", http.StatusOK)
		return
	case "push":
		push := &pushData{}

		json.Unmarshal(body, push)

		var (
			tok string
			err error
		)

		h.Log.Debug.Println("getting user and access token, provider_user_id =", push.Repo.Owner.ID)

		u, providerId, tok, err = h.getUserAndToken("github", push.Repo.Owner.ID)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		h.Log.Debug.Println("getting manifest urls from repository", push.Repo.ContentsURL, "at ref", push.HeadCommit.ID)

		rawurl := strings.Replace(push.Repo.ContentsURL, "{+path}", ".thrall", 1)

		urls, err := h.getManifestURLs(tok, rawurl+"?ref="+push.HeadCommit.ID, geturl)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		urls = append(urls, rawurl+".yml?ref="+push.HeadCommit.ID)

		h.Log.Debug.Println("loading manifests")

		mm, manifestErr = h.loadManifests(decodeBase64JSONManifest, tok, urls)

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

		var (
			tok string
			err error
		)

		if _, ok := actions[pull.Action]; !ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		h.Log.Debug.Println("getting user and access token, provider_user_id =", pull.PullRequest.User.ID)

		u, providerId, tok, err = h.getUserAndToken("github", pull.PullRequest.User.ID)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		h.Log.Debug.Println("getting manifest urls from repository", pull.PullRequest.Head.Repo.ContentsURL, "at ref", pull.PullRequest.Head.Sha)

		rawurl := strings.Replace(pull.PullRequest.Head.Repo.ContentsURL, "{+path}", ".thrall", 1)

		urls, err := h.getManifestURLs(tok, rawurl+"?ref="+pull.PullRequest.Head.Sha, geturl)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		urls = append(urls, rawurl+".yml?ref="+pull.PullRequest.Head.Sha)

		h.Log.Debug.Println("loading manifests")

		mm, manifestErr = h.loadManifests(decodeBase64JSONManifest, tok, urls)

		action := pull.Action

		if action == "synchronize" {
			action = "synchronized"
		}

		t.Type = build.Pull
		t.Comment = pull.PullRequest.Title
		t.Data.Set("id", strconv.FormatInt(pull.Number, 10))
		t.Data.Set("url", pull.PullRequest.URL)
		t.Data.Set("ref", pull.PullRequest.Base.Ref)
		t.Data.Set("username", pull.PullRequest.User.Login)
		t.Data.Set("action", action)
	}

	t.ProviderID = sql.NullInt64{
		Int64: providerId,
		Valid: true,
	}

	h.Log.Debug.Println("submitting", len(mm), "build manifests")

	if err := h.submitBuilds(mm, u, t); err != nil {
		cause := errors.Cause(err)

		if cause == namespace.ErrName {
			web.Text(w, invalidNamespaceName, http.StatusBadRequest)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.Text(w, "Failed to create build: "+cause.Error(), http.StatusInternalServerError)
		return
	}

	if manifestErr != nil {
		h.Log.Debug.Println("found some invalid manifests, responding with 202 Accepted")

		web.Text(w, invalidManifest+"\n\n"+manifestErr.Error(), http.StatusAccepted)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Gitlab is the webhook for the "Push Hook", and "Merge Request Hook" events.
// Before each event is handled individually the request'x X-Gitlab-Token is
// checked to make sure it is valid. On a successful handling of an event this
// send back a 204 No Content, otherwise we send back a plain text response with
// 500 Internal Server Error, and the verbatim error cause in the response body.
// Detailed below is how each event is handled,
//
// Push Hook - This will extract information about the author of the pushed
// commit and use this to create a build.Push build.Trigger. The data of the
// trigger will contain the id of the head commit, the url of the head commit,
// and the ref. The email, and username keys will be populated from the author
// of the commit. The build manifests are then extracted from the repository
// that was pushed to.
//
// Merge Request Hook -  This will created a build.Pull build.Trigger. The data
// of the trigger will contain the id of the merge request, the url of the merge
// request, and the ref to the merge request. The author's username of the
// merge request is set in the trigger's data. The build manifests are then
// extracted from the head repository of the merge request.
func (h Hook) Gitlab(w http.ResponseWriter, r *http.Request) {
	h.Log.Debug.Println("gitlab webhook received")

	event := r.Header.Get("X-Gitlab-Event")
	token := r.Header.Get("X-Gitlab-Token")

	if !bytes.Equal([]byte(token), h.Oauth2Providers["gitlab"].Secret()) {
		web.Text(w, "Invalid request token\n", http.StatusForbidden)
		return
	}

	var (
		manifestErr error
		providerId  int64
		mm          []config.Manifest
		u           *user.User
		t           *build.Trigger
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
		Commits []struct {
			ID      string
			Message string
			URL     string
			Author  map[string]string
		}
	}

	type mergeData struct {
		User struct {
			Name string
		}
		Project repo
		Attrs   struct {
			ID              int64  `json:"iid"`
			TargetBranch    string `json:"target_branch"`
			SourceProjectID int64  `json:"source_project_id"`
			AuthorID        int64  `json:"author_id"`
			Title           string
			Source          struct {
				WebURL string `json:"web_url"`
			}
			LastCommit struct {
				ID string
			} `json:"last_commit"`
			URL    string
			Action string
		} `json:"object_attributes"`
	}

	h.Log.Debug.Println("gitlab event type:", event)

	geturl := func(rawurl, ref string) func(map[string]string) string {
		return func(m map[string]string) string {
			if m["type"] == "tree" {
				return ""
			}

			return rawurl + "/repository/files/" + url.QueryEscape(m["path"]) + "?ref=" + ref
		}
	}

	switch event {
	case "Push Hook":
		push := &pushData{}

		json.NewDecoder(r.Body).Decode(push)

		var (
			tok string
			err error
		)

		h.Log.Debug.Println("getting user and access token, provider_user_id =", push.UserID)

		u, providerId, tok, err = h.getUserAndToken("github", push.UserID)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		head := push.Commits[len(push.Commits)-1]

		h.Log.Debug.Println("getting manifest urls from repository", push.Project.WebURL, "at ref", head.ID)

		url, _ := url.Parse(push.Project.WebURL)

		rawurl := url.Scheme + "://" + url.Host + "/api/v4/projects/" + strconv.FormatInt(push.Project.ID, 10) + "/repository/files/.thrall"

		urls, err := h.getManifestURLs(tok, rawurl+"?ref="+head.ID, geturl(rawurl, head.ID))

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		urls = append(urls, rawurl+".yml?ref="+head.ID)

		h.Log.Debug.Println("loading manifests")

		mm, manifestErr = h.loadManifests(decodeBase64JSONManifest, tok, urls)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
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

		var (
			tok string
			err error
		)

		h.Log.Debug.Println("getting user and access token, provider_user_id =", merge.Attrs.AuthorID)

		u, providerId, tok, err = h.getUserAndToken("github", merge.Attrs.AuthorID)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		h.Log.Debug.Println("getting manifest urls from repository", merge.Attrs.Source.WebURL, "at ref", merge.Attrs.LastCommit.ID)

		url, _ := url.Parse(merge.Project.WebURL)

		rawurl := url.Scheme + "://" + url.Host + "/api/v4/projects/" + strconv.FormatInt(merge.Attrs.SourceProjectID, 10) + "/repository/files/.thrall"

		urls, err := h.getManifestURLs(tok, rawurl, geturl(rawurl, merge.Attrs.LastCommit.ID))

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
			web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
			return
		}

		urls = append(urls, rawurl+".yml?ref="+merge.Attrs.LastCommit.ID)

		h.Log.Debug.Println("loading manifests")

		mm, manifestErr = h.loadManifests(decodeBase64JSONManifest, tok, urls)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(errors.Err(err)))
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

	t.ProviderID = sql.NullInt64{
		Int64: providerId,
		Valid: true,
	}

	h.Log.Debug.Println("submitting", len(mm), "build manifests")

	if err := h.submitBuilds(mm, u, t); err != nil {
		cause := errors.Cause(err)

		if cause == namespace.ErrName {
			web.Text(w, invalidNamespaceName, http.StatusBadRequest)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.Text(w, "Failed to create build: "+cause.Error(), http.StatusInternalServerError)
		return
	}

	if manifestErr != nil {
		web.Text(w, invalidManifest+"\n\n"+manifestErr.Error(), http.StatusAccepted)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
