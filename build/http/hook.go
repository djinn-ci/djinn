package http

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"djinn-ci.com/build"
	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/manifest"
	"djinn-ci.com/namespace"
	"djinn-ci.com/provider"
	"djinn-ci.com/provider/github"
	"djinn-ci.com/provider/gitlab"
	"djinn-ci.com/runner"
	"djinn-ci.com/server"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

type Hook struct {
	*server.Server

	Builds *build.Store
}

var (
	invalidManifest = `The hook was accepted, however some of the manifests retrieved from the
repository appeared to contain invalid YAML, see below:`

	invalidNamespaceName = `Invalid namespace name in build manifest, the name can only contain letters or
numbers`
)

// hookData represents the minimum data extracted from a webhook's event payload
// we need to submit builds.
type hookData struct {
	userId  int64  // the id of the user from the provider who sent the hook
	repoId  int64  // the id of the repo from the provider
	dirurl  string // the url in the repo that can contain multiple manifests, typically ".djinn"
	ref     string // the git ref we should use when retrieving manifests from the repo
	typ     build.TriggerType
	comment string
	data    map[string]string // additional information about the hook, such as build author
}

func (h *Hook) getUserRepoProvider(name string, repoId int64) (*user.User, *provider.Repo, *provider.Provider, error) {
	repos := provider.RepoStore{Pool: h.DB}

	r, _, err := repos.Get(
		query.Where("repo_id", "=", query.Arg(repoId)),
		query.Where("provider_name", "=", query.Arg(name)),
	)

	if err != nil {
		return nil, nil, nil, errors.Err(err)
	}

	providers := provider.Store{
		Pool:    h.DB,
		AESGCM:  h.AESGCM,
		Clients: h.Providers,
	}

	p, _, err := providers.Get(query.Where("id", "=", query.Arg(r.ProviderID)))

	if err != nil {
		return nil, nil, nil, errors.Err(err)
	}

	users := user.Store{Pool: h.DB}

	u, _, err := users.Get(query.Where("id", "=", query.Arg(p.UserID)))

	if err != nil {
		return nil, nil, nil, errors.Err(err)
	}
	return u, r, p, nil
}

func (h *Hook) getManifestURLs(tok, rawurl string, geturl func(map[string]string) string) ([]string, error) {
	urls := make([]string, 0)

	url, _ := url.Parse(rawurl)

	var cli http.Client

	resp, err := cli.Do(&http.Request{
		Method: "GET",
		URL:    url,
		Header: http.Header{
			"Authorization": {"Bearer " + tok},
		},
	})

	if err != nil {
		return nil, errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		h.Log.Debug.Println("no manifest urls found")
		return nil, nil
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

func (h *Hook) loadManifests(decode manifestDecoder, tok string, urls []string) ([]manifest.Manifest, error) {
	errs := make(chan error)
	manifests := make(chan manifest.Manifest)

	var wg sync.WaitGroup
	wg.Add(len(urls))

	for _, rawurl := range urls {
		go func(wg *sync.WaitGroup, rawurl string) {
			defer wg.Done()

			url, _ := url.Parse(rawurl)

			var cli http.Client

			resp, err := cli.Do(&http.Request{
				Method: "GET",
				URL:    url,
				Header: http.Header{
					"Authorization": {"Bearer " + tok},
				},
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
				errs <- errors.New("unexpected http status for " + rawurl + ": " + resp.Status)
				return
			}

			parts := strings.Split(strings.Split(rawurl, "?")[0], "/")
			path := strings.Join(parts[len(parts)-1:], "/")

			m, err := decode(resp.Body)

			if err != nil {
				errs <- errors.New("failed to decode manifest " + path + ": " + errors.Cause(err).Error())
				return
			}

			if err := m.Validate(); err != nil {
				errs <- errors.New("manifest invalid: " + path + ": " + err.Error())
				return
			}
			manifests <- m
		}(&wg, rawurl)
	}

	go func() {
		wg.Wait()
		close(errs)
		close(manifests)
	}()

	mm := make([]manifest.Manifest, 0, len(urls))
	msgs := make([]string, 0, len(urls))

	for errs != nil && manifests != nil {
		select {
		case err, ok := <-errs:
			if !ok {
				errs = nil
				break
			}
			msgs = append(msgs, err.Error())
		case m, ok := <-manifests:
			if !ok {
				manifests = nil
				break
			}
			mm = append(mm, m)
		}
	}

	if len(msgs) > 0 {
		return nil, errors.Slice(msgs)
	}
	return mm, nil
}

type manifestDecoder func(io.Reader) (manifest.Manifest, error)

func decodeBase64JSONManifest(r io.Reader) (manifest.Manifest, error) {
	var m manifest.Manifest

	var file struct {
		Encoding string
		Content  string
	}

	if err := json.NewDecoder(r).Decode(&file); err != nil {
		return m, errors.Err(err)
	}

	if file.Encoding != "base64" {
		return m, errors.New("unexpected file encoding: " + file.Encoding)
	}

	raw, err := base64.StdEncoding.DecodeString(file.Content)

	if err != nil {
		return m, errors.Err(err)
	}

	m, err = manifest.Unmarshal([]byte(raw))

	if err != nil {
		return m, errors.Err(err)
	}
	return m, nil
}

func (h *Hook) submitBuilds(ctx context.Context, mm []manifest.Manifest, host string, u *user.User, t *build.Trigger) ([]*build.Build, error) {
	bb := make([]*build.Build, 0, len(mm))

	var tag string

	if t.Type == build.Push || t.Type == build.Pull {
		tag = t.Data["ref"]
	}

	for _, m := range mm {
		b, err := h.Builds.Create(build.Params{
			UserID:   u.ID,
			Trigger:  t,
			Manifest: m,
			Tags:     []string{tag},
		})

		if err != nil {
			return nil, errors.Err(err)
		}
		bb = append(bb, b)
	}

	for _, b := range bb {
		if err := h.Builds.Submit(ctx, host, b); err != nil {
			return nil, errors.Err(err)
		}
	}
	return bb, nil
}

func (h *Hook) executeHook(ctx context.Context, host, name string, data hookData, geturl func(map[string]string) string) error {
	h.Log.Debug.Println("data.userId =", data.userId)

	u, r, p, err := h.getUserRepoProvider(name, data.repoId)

	if err != nil {
		return errors.Err(err)
	}

	b, err := h.AESGCM.Decrypt(p.AccessToken)

	if err != nil {
		return errors.Err(err)
	}

	tok := string(b)

	dirurl := data.dirurl + "?ref=" + data.ref

	urls, err := h.getManifestURLs(tok, dirurl, geturl)

	if err != nil {
		return errors.Err(err)
	}

	urls = append(urls, data.dirurl+".yml?ref="+data.ref)

	mm, manifesterr := h.loadManifests(decodeBase64JSONManifest, tok, urls)

	t := &build.Trigger{
		ProviderID: sql.NullInt64{
			Int64: p.ID,
			Valid: true,
		},
		RepoID: sql.NullInt64{
			Int64: r.ID,
			Valid: true,
		},
		Type:    data.typ,
		Comment: data.comment,
		Data:    data.data,
	}

	h.Log.Debug.Println("submitting", len(mm), "build manifests")

	bb, err := h.submitBuilds(ctx, mm, host, u, t)

	if err != nil {
		return errors.Err(err)
	}

	if t.Type == build.Pull {
		last := bb[len(bb)-1]
		last.User = u

		if err := p.SetCommitStatus(r, runner.Queued, host+last.Endpoint(), data.ref); err != nil {
			return errors.Err(err)
		}
	}
	return manifesterr
}

func getGithubURL(item map[string]string) string {
	if item["type"] == "dir" {
		return ""
	}

	if strings.HasSuffix(item["name"], ".yml") {
		return item["url"]
	}
	return ""
}

// Github serves the response for webhooks received from GitHub for the "ping",
// "push", and "pull_request" events. The owner of each build submitted will be
// whoever the access token being used to make API requests belongs to.
//
// This will respond with 204 No Content on a successful execution of a hook,
// otherwise it will respond with 500 Internal Server Error. If any invalid
// manifest if found during execution, then the response 202 Accepted is sent,
// with a message detailing what was wrong. If an invalid namespace name is
// found in any of the manifests, then 400 Bad Request is sent.
//
// On a "push" event, the repository being pushed to will be scraped for all
// manifests. This will first attempt to get the .djinn.yml file in the root,
// then any *.yml file found in the .djinn directory. The commit message is
// used as the build comment.
//
// On a "pull_request" event, the repository from which the pull request was
// made is scraped for manifests. The title and body of the pull request are
// used as the build comment.
func (h *Hook) Github(w http.ResponseWriter, r *http.Request) {
	h.Log.Debug.Println("github webhook received")

	event := r.Header.Get("X-Github-Event")

	cli, err := h.Providers.Get("github")

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	body, err := cli.VerifyRequest(r.Body, r.Header.Get("X-Hub-Signature"))

	if err != nil {
		h.Error(w, r, errors.Cause(err).Error(), http.StatusBadRequest)
		return
	}

	h.Log.Debug.Println("github event type:", event)

	var data hookData

	switch event {
	case "ping":
		webutil.Text(w, "pong\n", http.StatusOK)
		return
	case "push":
		data.typ = build.Push

		var push github.PushEvent
		json.Unmarshal(body, &push)

		data.userId = push.Repo.Owner.ID
		data.repoId = push.Repo.ID
		data.dirurl = strings.Replace(push.Repo.ContentsURL, "{+path}", ".djinn", 1)
		data.ref = push.HeadCommit.ID
		data.comment = push.HeadCommit.Message
		data.data = map[string]string{
			"repo_id":  strconv.FormatInt(push.Repo.ID, 10),
			"provider": "github",
			"url":      push.HeadCommit.URL,
			"ref":      push.Ref,
			"sha":      push.HeadCommit.ID,
			"email":    push.HeadCommit.Author["email"],
			"username": push.HeadCommit.Author["username"],
		}
	case "pull_request":
		data.typ = build.Pull

		var pull github.PullRequestEvent
		json.Unmarshal(body, &pull)

		if !github.IsValidPullRequestAction(pull.Action) {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		data.userId = pull.PullRequest.Base.User.ID
		data.repoId = pull.PullRequest.Base.Repo.ID
		data.dirurl = strings.Replace(pull.PullRequest.Head.Repo.ContentsURL, "{+path}", ".djinn", 1)
		data.ref = pull.PullRequest.Head.Sha
		data.comment = pull.PullRequest.Title

		action := pull.Action

		if action == "synchronize" {
			action = "synchronized"
		}

		data.data = map[string]string{
			"repo_id":  strconv.FormatInt(pull.PullRequest.Base.Repo.ID, 10),
			"provider": "github",
			"id":       strconv.FormatInt(pull.Number, 10),
			"url":      pull.PullRequest.HTMLURL,
			"ref":      pull.PullRequest.Base.Ref,
			"sha":      pull.PullRequest.Head.Sha,
			"username": pull.PullRequest.User.Login,
			"action":   action,
		}
	}

	if err := h.executeHook(r.Context(), webutil.BaseAddress(r), "github", data, getGithubURL); err != nil {
		cause := errors.Cause(err)

		if errs, ok := cause.(errors.Slice); ok {
			h.Log.Debug.Println("found some invalid manifests, responding with 202 Accepted")
			webutil.Text(w, invalidManifest+"\n\n"+errs.Error(), http.StatusAccepted)
			return
		}

		if derr, ok := cause.(*driver.Error); ok {
			webutil.Text(w, derr.Error(), http.StatusBadRequest)
			return
		}

		if cause == namespace.ErrName {
			webutil.Text(w, invalidNamespaceName, http.StatusBadRequest)
			return
		}
		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func getGitLabURL(rawurl, ref string) func(map[string]string) string {
	return func(item map[string]string) string {
		if item["type"] == "tree" {
			return ""
		}

		if path := item["path"]; strings.HasSuffix(path, ".yml") {
			return rawurl + "/repository/files/" + url.QueryEscape(path) + "?ref=" + ref
		}
		return ""
	}
}

func (h *Hook) Gitlab(w http.ResponseWriter, r *http.Request) {
	h.Log.Debug.Println("gitlab webhook received")

	event := r.Header.Get("X-Gitlab-Event")

	cli, err := h.Providers.Get("gitlab")

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	body, err := cli.VerifyRequest(r.Body, r.Header.Get("X-Gitlab-Token"))

	if err != nil {
		webutil.Text(w, errors.Cause(err).Error(), http.StatusBadRequest)
		return
	}

	h.Log.Debug.Println("gitlab event type:", event)

	gl, ok := cli.(*gitlab.Client)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to type assert provider.Client to *gitlab.Client")
		h.Error(w, r, "failed to type assert provider.Client to *gitlab.Client", http.StatusInternalServerError)
		return
	}

	var data hookData

	switch event {
	case "Push Hook":
		data.typ = build.Push

		var push gitlab.PushEvent
		json.Unmarshal(body, &push)

		if len(push.Commits) == 0 {
			webutil.Text(w, "No new commits", http.StatusAccepted)
			return
		}

		head := push.Commits[0]

		data.userId = push.UserID
		data.repoId = push.Project.ID
		data.dirurl = gl.APIEndpoint + "/projects/" + strconv.FormatInt(push.Project.ID, 10) + "/repository/files/.djinn"
		data.ref = head.ID
		data.comment = head.Message
		data.data = map[string]string{
			"repo_id":  strconv.FormatInt(push.Project.ID, 10),
			"provider": "gitlab",
			"url":      head.URL,
			"ref":      push.Ref,
			"sha":      head.ID,
			"email":    head.Author["email"],
			"username": head.Author["name"],
		}
	case "Merge Request Hook":
		actions := map[string]string{
			"merge":  "merged",
			"open":   "opened",
			"update": "updated",
		}

		data.typ = build.Pull

		var merge gitlab.MergeRequestEvent
		json.Unmarshal(body, &merge)

		data.userId = merge.User.ID
		data.repoId = merge.Project.ID
		data.dirurl = gl.APIEndpoint + "/projects/" + strconv.FormatInt(merge.Attrs.SourceProjectID, 10) + "/repository/files/.djinn"
		data.ref = merge.Attrs.LastCommit.ID
		data.comment = merge.Attrs.Title
		data.data = map[string]string{
			"repo_id":  strconv.FormatInt(merge.Attrs.TargetProjectID, 10),
			"provider": "gitlab",
			"id":       strconv.FormatInt(merge.Attrs.ID, 10),
			"url":      merge.Attrs.URL,
			"ref":      merge.Attrs.TargetBranch,
			"sha":      merge.Attrs.LastCommit.ID,
			"username": merge.User.Username,
			"action":   actions[merge.Attrs.Action],
		}
	}

	geturl := getGitLabURL(data.dirurl, data.ref)

	if err := h.executeHook(r.Context(), webutil.BaseAddress(r), "gitlab", data, geturl); err != nil {
		cause := errors.Cause(err)

		if errs, ok := cause.(errors.Slice); ok {
			h.Log.Debug.Println("found some invalid manifests, responding with 202 Accepted")
			webutil.Text(w, invalidManifest+"\n\n"+errs.Error(), http.StatusAccepted)
			return
		}

		switch err := cause.(type) {
		case *namespace.PathError:
			webutil.Text(w, err.Error(), http.StatusBadRequest)
			return
		case *gitlab.Error:
			webutil.Text(w, err.Error(), err.StatusCode)
			return
		case *driver.Error:
			webutil.Text(w, err.Error(), http.StatusBadRequest)
			return
		}

		if cause == namespace.ErrName {
			webutil.Text(w, invalidNamespaceName, http.StatusBadRequest)
			return
		}
		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func RegisterHooks(srv *server.Server) {
	hook := Hook{
		Server: srv,
		Builds: &build.Store{
			Pool:         srv.DB,
			Hasher:       srv.Hasher,
			DriverQueues: srv.DriverQueues,
		},
	}

	sr := srv.Router.PathPrefix("/hook").Subrouter()
	sr.HandleFunc("/github", hook.Github)
	sr.HandleFunc("/gitlab", hook.Gitlab)
}
